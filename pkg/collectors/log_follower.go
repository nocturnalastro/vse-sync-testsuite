// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/contexts"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

const (
	lineSliceChanLength = 100
	lineChanLength      = 1000
	lineDelim           = '\n'
	streamingBufferSize = 2000
	logPollInterval     = 2
	logFilePermissions  = 0666
	keepGenerations     = 5
)

var (
	followDuration = logPollInterval * time.Second
	followTimeout  = 30 * followDuration
)

type ProcessedLine struct {
	Timestamp  time.Time
	Raw        string
	Content    string
	Generation uint32
}

type GenerationalLockedTime struct {
	time       time.Time
	lock       sync.RWMutex
	generation uint32
}

func (lt *GenerationalLockedTime) Time() time.Time {
	lt.lock.RLock()
	defer lt.lock.RUnlock()
	return lt.time
}

func (lt *GenerationalLockedTime) Generation() uint32 {
	lt.lock.RLock()
	defer lt.lock.RUnlock()
	return lt.generation
}

func (lt *GenerationalLockedTime) Update(update time.Time) {
	lt.lock.Lock()
	defer lt.lock.Unlock()
	lt.time = update
	lt.generation += 1
}

type LineSlice struct {
	start      time.Time
	end        time.Time
	lines      []*ProcessedLine
	generation uint32
}

// LogsCollector collects logs from repeated calls to the kubeapi with overlapping query times,
// the lines are then fed into a channel, in another gorotine they are de-duplicated and written to an output file.
//
// Overlap:
// cmd       followDuration
// |---------------|
// since          cmd        followDuration
// |---------------|---------------|
// .............. since           cmd        followDuration
// ..............  |---------------|---------------|
//
// This was done because oc logs and the kubeapi endpoint which it uses does not look back
// over a log rotation event, nor does it continue to follow.
//
// Log aggregators would be preferred over this method however this requires extra infra
// which might not be present in the environment.
type LogsCollector struct {
	sliceQuit          chan os.Signal
	writeQuit          chan os.Signal
	lines              chan *ProcessedLine
	lineSlices         chan *LineSlice
	generations        map[uint32][]*LineSlice
	oldestGen          uint32
	client             *clients.Clientset
	logsOutputFileName string
	lastPoll           GenerationalLockedTime
	wg                 sync.WaitGroup
	pollInterval       int
	withTimeStamps     bool
	running            bool
	pruned             bool
}

var fileNameNumber int

const (
	LogsCollectorName = "Logs"
	LogsInfo          = "log-line"
)

func (logs *LogsCollector) GetPollInterval() int {
	return logs.pollInterval
}

func (logs *LogsCollector) IsAnnouncer() bool {
	return false
}

func (logs *LogsCollector) SetLastPoll(pollTime time.Time) {
	logs.lastPoll.Update(pollTime)
}

// Start sets up the collector so it is ready to be polled
func (logs *LogsCollector) Start() error {
	go logs.processSlices()
	go logs.writeToLogFile()
	logs.running = true
	return nil
}

func (logs *LogsCollector) consumeLineSlice(lineSlice *LineSlice) {
	logs.generations[lineSlice.generation] = append(logs.generations[lineSlice.generation], lineSlice)
}

func (logs *LogsCollector) writeLine(line *ProcessedLine, writer io.StringWriter) {
	var err error
	if logs.withTimeStamps {
		_, err = writer.WriteString(line.Raw + "\n")
	} else {
		_, err = writer.WriteString(line.Content + "\n")
	}
	if err != nil {
		log.Error(fmt.Errorf("failed to write log output to file"))
	}
}

func findOverlap(x, y []*ProcessedLine) int {
	// Start off by being dumb and just moving first line of the second
	position := len(x) - 1
	checkLine := y[0].Raw
	for position >= 0 {
		if x[position].Raw == checkLine {
			break
		}
		position--
	}
	return position
}

func checkOverlap(x, y []*ProcessedLine) bool {
	for i, line := range x {
		if line.Raw != y[i].Raw {
			return false
		}
	}
	return true
}

func writeOverlap(lines []*ProcessedLine, name string) error {
	fw, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("failed %w", err)
	}
	defer fw.Close()
	fileNameNumber++

	for _, line := range lines {
		fw.WriteString(line.Raw + "\n")
	}
	return nil
}

// func processOverlap(reference, other []*ProcessedLine) ([]*ProcessedLine, error) {
// 	// err := writeOverlap(reference)
// 	// if err != nil {
// 	// 	log.Error(err)
// 	// }
// 	// err = writeOverlap(other)
// 	// if err != nil {
// 	// 	log.Error(err)
// 	// }

// 	newRef, newOther, err := dedupWithoutCombine(reference, other)
// 	if err != nil {
// 		return reference, err
// 	}
// 	res := make([]*ProcessedLine, len(newRef)+len(newOther))
// 	res = append(res, newRef...)
// 	res = append(res, newOther...)
// 	return res, nil
// }

func dedupWithoutCombine(reference, other []*ProcessedLine) ([]*ProcessedLine, []*ProcessedLine, error) {
	position := findOverlap(reference, other)
	if position == -1 {
		return reference, other, nil
	}
	offset := len(reference) - position
	newOther := make([]*ProcessedLine, 0, len(other)-offset)
	newOther = append(newOther, other[offset:]...)

	if checkOverlap(reference[position:], other[:offset]) {
		return reference, newOther, nil
	}
	// TODO: attempt stitching here by dropping the failing line from the check
	// and keeping it to add in
	return reference, newOther, fmt.Errorf("dropping lines: overlapping log slices don't match this suggests missing lines, don't know how to combine")
}

// func dedupLineSlices(lineSlices []*LineSlice) *LineSlice {
// 	// Assuming there a no missing lines and that overlaps are continuus.
// 	// We can order the slices find the max overlap in the two
// 	// Then check for an overlap
// 	// remove the overlap from the second and append the rest
// 	// then keep taking the next LineSlice
// 	// until we have stiched them all together

// 	sort.Slice(lineSlices, func(i, j int) bool {
// 		startDiff := lineSlices[i].start.Sub(lineSlices[j].start)
// 		if startDiff == 0 {
// 			endDiff := lineSlices[i].start.Sub(lineSlices[j].start)
// 			return endDiff > 0
// 		}
// 		return startDiff < 0
// 	})

// 	reference := lineSlices[0].lines
// 	var err error
// 	for _, other := range lineSlices[1:] {
// 		reference, err = processOverlap(reference, other.lines)
// 		if err != nil {
// 			log.Error(err)
// 		}
// 	}
// 	return &LineSlice{
// 		lines: reference,
// 		start: reference[0].Timestamp,
// 		end:   reference[len(reference)-1].Timestamp,
// 	}
// }

func makeSliceFromLines(lines []*ProcessedLine) *LineSlice {
	return &LineSlice{
		lines: lines,
		start: lines[0].Timestamp,
		end:   lines[len(lines)-1].Timestamp,
	}
}

func dedupLineSlicesWithoutJoining(lineSlices []*LineSlice) [][]*ProcessedLine {
	// Assuming there a no missing lines and that overlaps are continuus.
	// We can order the slices find the max overlap in the two
	// Then check for an overlap
	// remove the overlap from the second and append the rest
	// then keep taking the next LineSlice
	// until we have stiched them all together

	for _, ls := range lineSlices {
		err := writeOverlap(ls.lines, fmt.Sprintf("ProcessOverlap%d.log", fileNameNumber))
		if err != nil {
			log.Error(err)
		}
	}

	if len(lineSlices) == 1 {
		return [][]*ProcessedLine{lineSlices[0].lines}
	}

	sort.Slice(lineSlices, func(i, j int) bool {
		startDiff := lineSlices[i].start.Sub(lineSlices[j].start)
		if startDiff == 0 {
			endDiff := lineSlices[i].start.Sub(lineSlices[j].start)
			return endDiff > 0
		}
		return startDiff < 0
	})

	reference := lineSlices[0].lines
	dedupedSliceSegments := make([][]*ProcessedLine, 0)

	for i, other := range lineSlices[1:] {
		segment, dedupledOther, err := dedupWithoutCombine(reference, other.lines)
		if err != nil {
			log.Error("dropling lines in second segment", err)
			// TODO handle this better:
			//   If the next segment starts at the same time as other
			//   then if we combine the dedup then it may cause issues
			//   so drop the lines and hope its not too much data
			continue
		}
		dedupedSliceSegments = append(dedupedSliceSegments, segment)
		// length of lineSlices[1:] -1
		if i == len(lineSlices)-2 {
			dedupedSliceSegments = append(dedupedSliceSegments, dedupledOther)
		} else {
			reference = dedupledOther
		}
	}

	return dedupedSliceSegments
}

func combineSliceSegmenets(segments ...[]*ProcessedLine) []*ProcessedLine {
	newLength := 0
	for _, s := range segments {
		newLength += len(s)
	}
	result := make([]*ProcessedLine, 0, newLength)
	for _, s := range segments {
		result = append(result, s...)
	}

	return result
}

func dedup(generationalLineSlices [][]*LineSlice) ([]*ProcessedLine, *LineSlice) {
	dedupedGenerations := make([]*LineSlice, len(generationalLineSlices))
	for i, gen := range generationalLineSlices {
		dedupedGenerations[i] = makeSliceFromLines(
			combineSliceSegmenets(
				dedupLineSlicesWithoutJoining(gen)...,
			),
		)
	}
	fullyDedup := dedupLineSlicesWithoutJoining(dedupedGenerations)

	writeLines := combineSliceSegmenets(fullyDedup[:len(fullyDedup)-1]...)
	log.Info("logs: dedupled lines", len(writeLines))
	return writeLines, makeSliceFromLines(fullyDedup[len(fullyDedup)-1])
}

func (logs *LogsCollector) flushGenerations(generations []uint32) {
	log.Info("logs: Flushing Generations ", generations)
	generationalLineSlices := make([][]*LineSlice, len(generations))
	for i, genIndex := range generations {
		gen, ok := logs.generations[genIndex]
		if !ok {
			log.Error("Go gen at this index", genIndex, logs.generations)
		}
		generationalLineSlices[i] = gen
		log.Info("Gen ", genIndex, gen)
	}

	writeLines, lastGen := dedup(generationalLineSlices)

	genlis := ""
	for i, genIndex := range generations {
		if i < len(generations)-1 {
			genlis += fmt.Sprintf("-%d", genIndex)
		}
	}
	err := writeOverlap(writeLines, fmt.Sprintf("Generations%s.log", genlis))
	log.Debug(err)

	for _, line := range writeLines {
		logs.lines <- line
	}

	logs.oldestGen = generations[len(generations)-1]
	logs.generations[logs.oldestGen] = []*LineSlice{lastGen}
	for i := 0; i < len(generations)-1; i++ {
		delete(logs.generations, generations[i])
	}
}

//nolint:cyclop // allow this to be a little complicated
func (logs *LogsCollector) processSlices() {
	logs.wg.Add(1)
	defer logs.wg.Done()
	var seenGeneration uint32 = 0
	tryFlush := false
	for {
		select {
		case sig := <-logs.sliceQuit:
			// Consume the rest of the lines so we don't miss lines
			for len(logs.lineSlices) > 0 {
				lineSlice := <-logs.lineSlices
				logs.consumeLineSlice(lineSlice)
			}

			gensToFlush := make([]uint32, len(logs.generations))
			i := 0
			for g := range logs.generations {
				gensToFlush[i] = g
				i++
			}
			logs.flushGenerations(gensToFlush)

			logs.writeQuit <- sig
			return
		case lineSlice := <-logs.lineSlices:
			if seenGeneration < lineSlice.generation {
				seenGeneration = lineSlice.generation
				tryFlush = true
			}
			logs.consumeLineSlice(lineSlice)
		default:
			if tryFlush {
				if seenGeneration-logs.oldestGen > keepGenerations {
					gensToFlush := make([]uint32, keepGenerations)
					for i := 0; i < keepGenerations; i++ {
						gensToFlush[i] = logs.oldestGen + uint32(i)
					}
					logs.flushGenerations(gensToFlush)
				}
				tryFlush = false
			} else {
				time.Sleep(time.Nanosecond)
			}
		}
	}
}

func (logs *LogsCollector) writeToLogFile() {
	logs.wg.Add(1)
	defer logs.wg.Done()

	fileHandle, err := os.OpenFile(logs.logsOutputFileName, os.O_CREATE|os.O_WRONLY, logFilePermissions)
	utils.IfErrorExitOrPanic(err)
	defer fileHandle.Close()
	for {
		select {
		case <-logs.writeQuit:
			// Consume the rest of the lines so we don't miss lines
			for len(logs.lines) > 0 {
				line := <-logs.lines
				logs.writeLine(line, fileHandle)
			}
			return
		case line := <-logs.lines:
			logs.writeLine(line, fileHandle)
		default:
			time.Sleep(time.Nanosecond)
		}
	}
}

func processLine(line string) (*ProcessedLine, error) {
	splitLine := strings.SplitN(line, " ", 2) //nolint:gomnd // moving this to a var would make the code less clear
	if len(splitLine) < 2 {                   //nolint:gomnd // moving this to a var would make the code less clear
		return nil, fmt.Errorf("failed to split line %s", line)
	}
	timestampPart := splitLine[0]
	lineContent := splitLine[1]
	timestamp, err := time.Parse(time.RFC3339, timestampPart)
	if err != nil {
		// This is not a value line something went wrong
		return nil, fmt.Errorf("failed to process timestamp from line: '%s'", line)
	}
	processed := &ProcessedLine{
		Timestamp: timestamp,
		Content:   lineContent,
		Raw:       line,
	}
	return processed, nil
}

func (logs *LogsCollector) processLines(line string, lineSlice *LineSlice) (string, time.Time) {
	var lastTimestamp time.Time
	if strings.ContainsRune(line, lineDelim) {
		lines := strings.Split(line, "\n")
		for index := 0; index < len(lines)-1; index++ {
			log.Debug("logs: lines: ", lines[index])
			processed, err := processLine(lines[index])
			if err != nil {
				log.Warning("logs: error when processing lines: ", err)
			} else {
				lineSlice.lines = append(lineSlice.lines, processed)
				lastTimestamp = processed.Timestamp
			}
		}
		line = lines[len(lines)-1]
	}
	return line, lastTimestamp
}

func durationPassed(first, current time.Time, duration time.Duration) bool {
	if first.IsZero() {
		return false
	}
	if current.IsZero() {
		return false
	}
	return duration <= current.Sub(first)
}

//nolint:funlen // allow long function
func processStream(logs *LogsCollector, stream io.ReadCloser, sinceTime time.Duration) error {
	line := ""
	lastTimestamp := time.Time{}
	firstTimestamp := time.Time{}
	timestamp := time.Time{}
	buf := make([]byte, streamingBufferSize)
	expectedDuration := sinceTime + followDuration

	lineSlice := &LineSlice{
		lines:      make([]*ProcessedLine, 0),
		generation: logs.lastPoll.Generation(),
	}

	for !durationPassed(firstTimestamp, lastTimestamp, expectedDuration) {
		nBytes, err := stream.Read(buf)
		if err == io.EOF { //nolint:errorlint // No need for Is or As check as this should just be EOF
			log.Warning("log stream ended unexpectedly, possible log rotation detected at ", lastTimestamp)
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading buffer: %w", err)
		}
		if nBytes == 0 {
			continue
		}
		line += string(buf[:nBytes])
		line, timestamp = logs.processLines(line, lineSlice)

		// set First legitimate timestamp
		if !timestamp.IsZero() {
			if firstTimestamp.IsZero() {
				firstTimestamp = timestamp
			}
			lastTimestamp = timestamp
		}
	}

	log.Info("logs: Finish stream")

	if firstTimestamp.IsZero() || lastTimestamp.IsZero() {
		return fmt.Errorf("zero timestamp after processing lines first(%v) or last (%s)", firstTimestamp, lastTimestamp)
	}

	lineSlice.start = firstTimestamp
	lineSlice.end = lastTimestamp
	logs.lineSlices <- lineSlice

	return nil
}

func (logs *LogsCollector) poll() error {
	podName, err := logs.client.FindPodNameFromPrefix(contexts.PTPNamespace, contexts.PTPPodNamePrefix)
	if err != nil {
		return fmt.Errorf("failed to poll: %w", err)
	}
	sinceTime := time.Since(logs.lastPoll.Time())
	sinceSeconds := int64(sinceTime.Seconds())

	podLogOptions := v1.PodLogOptions{
		SinceSeconds: &sinceSeconds,
		Container:    contexts.PTPContainer,
		Follow:       true,
		Previous:     false,
		Timestamps:   true,
	}
	podLogRequest := logs.client.K8sClient.CoreV1().
		Pods(contexts.PTPNamespace).
		GetLogs(podName, &podLogOptions).
		Timeout(followTimeout)
	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to poll when r: %w", err)
	}
	defer stream.Close()

	start := time.Now()
	err = processStream(logs, stream, sinceTime)
	if err != nil {
		return err
	}
	logs.SetLastPoll(start)
	return nil
}

// Poll collects log lines
func (logs *LogsCollector) Poll(resultsChan chan PollResult, wg *utils.WaitGroupCount) {
	defer func() {
		wg.Done()
	}()
	errorsToReturn := make([]error, 0)
	err := logs.poll()
	if err != nil {
		errorsToReturn = append(errorsToReturn, err)
	}
	resultsChan <- PollResult{
		CollectorName: LogsCollectorName,
		Errors:        errorsToReturn,
	}
}

// CleanUp stops a running collector
func (logs *LogsCollector) CleanUp() error {
	logs.running = false
	logs.sliceQuit <- os.Kill
	logs.wg.Wait()
	return nil
}

// Returns a new LogsCollector from the CollectionConstuctor Factory
func NewLogsCollector(constructor *CollectionConstructor) (Collector, error) {
	collector := LogsCollector{
		running:            false,
		client:             constructor.Clientset,
		sliceQuit:          make(chan os.Signal),
		writeQuit:          make(chan os.Signal),
		pollInterval:       logPollInterval,
		pruned:             true,
		lineSlices:         make(chan *LineSlice, lineSliceChanLength),
		lines:              make(chan *ProcessedLine, lineChanLength),
		generations:        make(map[uint32][]*LineSlice),
		lastPoll:           GenerationalLockedTime{time: time.Now().Add(-time.Second)}, // Stop initial since seconds from being 0 as its invalid
		withTimeStamps:     constructor.IncludeLogTimestamps,
		logsOutputFileName: constructor.LogsOutputFile,
	}
	return &collector, nil
}

func init() {
	// Make log opt in as in may lose some data.
	RegisterCollector(LogsCollectorName, NewLogsCollector, includeByDefault)
}
