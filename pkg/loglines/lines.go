// SPDX-License-Identifier: GPL-2.0-or-later

package loglines

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

type ProcessedLine struct {
	Timestamp  time.Time
	Full       string
	Content    string
	Generation uint32
}

func ProcessLine(line string) (*ProcessedLine, error) {
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
		Content:   strings.TrimRightFunc(lineContent, unicode.IsSpace),
		Full:      strings.TrimRightFunc(line, unicode.IsSpace),
	}
	return processed, nil
}

func NewGenerationalLockedTime(initialTime time.Time) GenerationalLockedTime {
	return GenerationalLockedTime{time: initialTime}
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
	Lines      []*ProcessedLine
	Generation uint32
}

type Generations struct {
	Store  map[uint32][]*LineSlice
	Latest uint32
	Oldest uint32
}

func (gens *Generations) Add(lineSlice *LineSlice) {
	genSlice, ok := gens.Store[lineSlice.Generation]
	if !ok {
		genSlice = make([]*LineSlice, 0)
	}
	genSlice = append(genSlice, lineSlice)
	gens.Store[lineSlice.Generation] = genSlice

	log.Info("Logs: all generations: ", gens.Store)

	if gens.Latest < lineSlice.Generation {
		gens.Latest = lineSlice.Generation
		log.Info("Logs: lastest updated ", gens.Latest)
		log.Info("Logs: should flush ", gens.ShouldFlush())
	}
}

func (gens *Generations) removeOlderThan(keepGen uint32) {
	log.Info("Removing geners <", keepGen)
	for g := range gens.Store {
		if g < keepGen {
			delete(gens.Store, g)
		}
	}
	gens.Oldest = keepGen
}

func (gens *Generations) ShouldFlush() bool {
	return (gens.Latest-gens.Oldest > keepGenerations &&
		len(gens.Store) > keepGenerations)
}

func (gens *Generations) Flush() *LineSlice {
	lastGen := gens.Oldest + keepGenerations
	log.Info("Flushing generations <=", lastGen)

	gensToFlush := make([][]*LineSlice, 0)
	for index, value := range gens.Store {
		if index <= lastGen {
			gensToFlush = append(gensToFlush, value)
		}
	}
	result, lastSlice := gens.flush(gensToFlush)
	gens.removeOlderThan(lastSlice.Generation)
	gens.Store[lastSlice.Generation] = []*LineSlice{lastSlice}
	return result
}

func (gens *Generations) FlushAll() *LineSlice {
	log.Info("Flushing all generations")
	gensToFlush := make([][]*LineSlice, 0)
	for _, value := range gens.Store {
		gensToFlush = append(gensToFlush, value)
	}
	result, lastSlice := gens.flush(gensToFlush)
	return MakeSliceFromLines(MakeNewCombinedSlice(result.Lines, lastSlice.Lines), lastSlice.Generation)
}

//nolint:gocritic // don't want to name the return values as they should be built later
func (gens *Generations) flush(generations [][]*LineSlice) (*LineSlice, *LineSlice) {
	log.Info("genrations: ", generations)
	sort.Slice(generations, func(i, j int) bool {
		return generations[i][0].Generation < generations[j][0].Generation
	})
	dedupGen := make([]*LineSlice, len(generations))
	for index, gen := range generations {
		for j, g := range gen {
			err := WriteOverlap(g.Lines, fmt.Sprintf("Generation%d-%d-%d.log", g.Generation, j, fileNameNumber))
			if err != nil {
				log.Error(err)
			}
		}
		dedupGen[index] = dedupGeneration(gen)
		err := WriteOverlap(dedupGen[index].Lines, fmt.Sprintf("DedupGeneration%d-%d.log", gen[0].Generation, fileNameNumber))
		if err != nil {
			log.Error(err)
		}
	}
	fileNameNumber++
	return DedupLineSlices(dedupGen)
}

func MakeSliceFromLines(lines []*ProcessedLine, generation uint32) *LineSlice {
	return &LineSlice{
		Lines:      lines,
		start:      lines[0].Timestamp,
		end:        lines[len(lines)-1].Timestamp,
		Generation: generation,
	}
}
