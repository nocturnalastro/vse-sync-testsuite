// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

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

var fileNameNumber int
var overlapFile int

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

type Generations struct {
	store     map[uint32][]*LineSlice
	latest    uint32
	oldest    uint32
	reference *LineSlice
}

func (gens *Generations) Add(ls *LineSlice) {
	genSlice, ok := gens.store[ls.generation]
	if !ok {
		genSlice = make([]*LineSlice, 0)
	}
	genSlice = append(genSlice, ls)
	gens.store[ls.generation] = genSlice

	log.Info("all generations: ", gens.store)

	if gens.latest < ls.generation {
		gens.latest = ls.generation
		log.Info("Logs: lastest updated", gens.latest)
		log.Info("Logs: should flush", gens.ShouldFlush())
	}
}

func (gens *Generations) removeOlderThan(keepGen uint32) {
	for g := range gens.store {
		if g < keepGen {
			delete(gens.store, g)
		}
	}
	gens.oldest = keepGen
}

func (gens *Generations) ShouldFlush() bool {
	return (gens.latest-gens.oldest > keepGenerations &&
		len(gens.store) > keepGenerations)
}

func (gens *Generations) Flush() *LineSlice {
	lastGen := gens.oldest + keepGenerations
	log.Info("Flushing generations <=", lastGen)

	gensToFlush := make([][]*LineSlice, 0)
	for index, value := range gens.store {
		if index <= lastGen {
			gensToFlush = append(gensToFlush, value)
		}
	}
	result, lastSlice := gens.flush(gensToFlush)
	gens.removeOlderThan(lastSlice.generation)
	gens.store[lastSlice.generation] = []*LineSlice{lastSlice}
	return result
}

func (gens *Generations) FlushAll() *LineSlice {
	log.Info("Flushing all generations")
	gensToFlush := make([][]*LineSlice, 0)
	for _, value := range gens.store {
		gensToFlush = append(gensToFlush, value)
	}
	result, lastSlice := gens.flush(gensToFlush)
	return makeSliceFromLines(makeNewCombinedSlice(result.lines, lastSlice.lines), lastSlice.generation)
}

func (gens *Generations) flush(generations [][]*LineSlice) (*LineSlice, *LineSlice) {
	log.Info("genrations: ", generations)
	sort.Slice(generations, func(i, j int) bool {
		return generations[i][0].generation < generations[j][0].generation
	})
	dedupGen := make([]*LineSlice, len(generations))
	for index, gen := range generations {
		dedupGen[index] = dedupGeneration(gen)
	}
	return dedup(dedupGen)
}

func dedupGeneration(lineSlices []*LineSlice) *LineSlice {
	ls1, ls2 := dedup(lineSlices)
	return makeSliceFromLines(makeNewCombinedSlice(ls1.lines, ls2.lines), ls1.generation)
}

// findLineIndex will find the index of a line in a slice of lines
// and will return -1 if it is not found
func findLineIndex(list []*ProcessedLine, needle *ProcessedLine) int {
	checkLine := strings.TrimRightFunc(needle.Raw, unicode.IsSpace)
	for i, hay := range list {
		if strings.TrimRightFunc(hay.Raw, unicode.IsSpace) == checkLine {
			return i
		}
	}
	return -1
}

func dedupAB(a, b []*ProcessedLine) ([]*ProcessedLine, []*ProcessedLine) {
	firstLineIndex := findLineIndex(a, b[0])
	log.Info("line index: ", firstLineIndex)
	if firstLineIndex == -1 {
		log.Error("FUCK")
	}

	for i, line := range a[firstLineIndex:] {
		if strings.TrimRightFunc(line.Raw, unicode.IsSpace) != strings.TrimRightFunc(b[i].Raw, unicode.IsSpace) {
			log.Error("FUCK line match")
			fmt.Println(strings.TrimRightFunc(line.Raw, unicode.IsSpace))
			fmt.Println(strings.TrimRightFunc(b[i].Raw, unicode.IsSpace))
			fmt.Println(strings.TrimRightFunc(b[i+1].Raw, unicode.IsSpace))
			os.Exit(-1)
		}
	}

	return a[:firstLineIndex], b
}

func makeNewCombinedSlice(x, y []*ProcessedLine) []*ProcessedLine {
	r := make([]*ProcessedLine, 0, len(x)+len(y))
	r = append(r, x...)
	r = append(r, y...)
	return r
}

func dedup(lineSlices []*LineSlice) (*LineSlice, *LineSlice) {
	if len(lineSlices) == 1 {
		return &LineSlice{}, lineSlices[0]
	} else if len(lineSlices) == 2 {
		return lineSlices[0], lineSlices[1]
	}

	lastLineSlice := lineSlices[len(lineSlices)-1]
	lastButOneLineSlice := lineSlices[len(lineSlices)-2]

	// work backwards thought the slices
	// dedupling the earlier one along the way
	b, lastLines := dedupAB(lastButOneLineSlice.lines, lastLineSlice.lines)
	resLines := b
	reference := makeNewCombinedSlice(b, lastLines)

	for index := len(lineSlices) - 3; index >= 0; index-- {
		aLines, bLines := dedupAB(lineSlices[index].lines, reference)
		resLines = makeNewCombinedSlice(aLines, resLines)
		reference = makeNewCombinedSlice(aLines, bLines)
	}
	return makeSliceFromLines(resLines, lastButOneLineSlice.generation), makeSliceFromLines(lastLines, lastLineSlice.generation)
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

func makeSliceFromLines(lines []*ProcessedLine, generation uint32) *LineSlice {
	return &LineSlice{
		lines:      lines,
		start:      lines[0].Timestamp,
		end:        lines[len(lines)-1].Timestamp,
		generation: generation,
	}
}

//nolint:cyclop // allow this to be a little complicated
func (logs *LogsCollector) processSlices() {
	logs.wg.Add(1)
	defer logs.wg.Done()
	generations := Generations{
		store:  make(map[uint32][]*LineSlice),
		oldest: 0,
	}
	for {
		select {
		case sig := <-logs.sliceQuit:
			for len(logs.lines) > 0 {
				lineSlice := <-logs.lineSlices
				generations.Add(lineSlice)
			}
			generations.FlushAll()
			logs.writeQuit <- sig
			return
		case lineSlice := <-logs.lineSlices:
			generations.Add(lineSlice)
		default:
			if generations.ShouldFlush() {
				deduplicated := generations.Flush()
				for _, line := range deduplicated.lines {
					logs.lines <- line
				}
			}
			time.Sleep(time.Nanosecond)
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
func processStream(stream io.ReadCloser, expectedEndtime time.Time) ([]*ProcessedLine, error) {
	scanner := bufio.NewScanner(stream)
	segment := make([]*ProcessedLine, 0)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return segment, err
		}
		pline, err := processLine(scanner.Text())
		if err != nil {
			log.Warning("failed to process line: ", err)
			continue
		}
		segment = append(segment, pline)
		if expectedEndtime.Sub(pline.Timestamp) < 0 {
			// Were past our expected end time lets finish there
			break
		}
	}
	return segment, nil
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
	generation := logs.lastPoll.Generation()
	lines, err := processStream(stream, time.Now().Add(followDuration))
	if err != nil {
		return err
	}
	lineSlice := makeSliceFromLines(lines, generation)
	logs.lineSlices <- lineSlice
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
