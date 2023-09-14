// SPDX-License-Identifier: GPL-2.0-or-later

package loglines_test

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/loglines"
)

//nolint:unparam // its only one param for now but we might want more later
func loadLinesFromFile(path string, generation uint32) (*loglines.LineSlice, error) {
	reader, err := os.Open(path)
	if err != nil {
		return &loglines.LineSlice{}, fmt.Errorf("failed to open file %s %w", path, err)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)

	lines := make([]*loglines.ProcessedLine, 0)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return &loglines.LineSlice{}, fmt.Errorf("failed to read line from %s %w", path, err)
		}
		line, err := loglines.ProcessLine(scanner.Text())
		if err != nil {
			return &loglines.LineSlice{}, fmt.Errorf("failed to process line from %s %w", path, err)
		}
		lines = append(lines, line)
	}
	return loglines.MakeSliceFromLines(lines, generation), nil
}

var _ = Describe("NewContainerContext", func() {
	When("called Deduplicating a two line slices with total overlap", func() { //nolint:dupl // don't want to dupl tests
		It("should return a single set of the lines", func() {
			// TODO: Make setup failures distinct from test
			lineSlice1, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			lineSlice2, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			dl1, dl2 := loglines.DedupAB(lineSlice1.Lines, lineSlice2.Lines)
			Expect(dl1).To(BeEmpty())
			Expect(dl2).To(Equal(lineSlice1.Lines))
		})
	})
	When("called Deduplicating a two line slices with no overlap", func() { //nolint:dupl // don't want to dupl tests
		It("should return a both sets of the lines", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			dl1, dl2 := loglines.DedupAB(lineSlice.Lines[:100], lineSlice.Lines[200:300])
			Expect(dl1).To(Equal(lineSlice.Lines[:100]))
			Expect(dl2).To(Equal(lineSlice.Lines[200:300]))
		})
	})
	When("called Deduplicating a two line slices with some overlap", func() { //nolint:dupl // don't want to dupl tests
		It("should return a both sets of the lines", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			dl1, dl2 := loglines.DedupAB(lineSlice.Lines[:200], lineSlice.Lines[100:300])
			Expect(dl1).To(Equal(lineSlice.Lines[:100]))
			Expect(dl2).To(Equal(lineSlice.Lines[100:300]))
		})
	})
	When("called Deduplicating a if one is an extension of the other", func() { //nolint:dupl // don't want to dupl tests
		It("should return a empty set and a complete set should be return", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			dl1, dl2 := loglines.DedupAB(lineSlice.Lines[:200], lineSlice.Lines[:300])
			Expect(dl1).To(BeEmpty())
			Expect(dl2).To(Equal(lineSlice.Lines[:300]))
		})
	})
	When("called Deduplicating a two line slices with first line of a missing with overlap", func() { //nolint:dupl // don't want to dupl tests
		It("should return a empty set and a complete set should be return", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			dl1, dl2 := loglines.DedupAB(lineSlice.Lines[1:200], lineSlice.Lines[:300])
			Expect(dl1).To(BeEmpty())
			Expect(dl2).To(Equal(lineSlice.Lines[:300]))
		})
	})
	When("called Deduplicating a two line slices with first slice is missing every 3rd line", func() { //nolint:dupl // don't want to dupl tests
		It("should return a empty set and a complete set should be return", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			firstSet := make([]*loglines.ProcessedLine, 0)
			for i, line := range lineSlice.Lines[:200] {
				if i%3 == 0 {
					continue
				}
				firstSet = append(firstSet, line)
			}
			dl1, dl2 := loglines.DedupAB(firstSet, lineSlice.Lines[:300])
			Expect(dl1).To(BeEmpty())
			Expect(dl2).To(Equal(lineSlice.Lines[:300]))
		})
	})
	When("called Deduplicating a two line slices with second slice is missing every 3rd line", func() { //nolint:dupl // don't want to dupl tests
		It("should return a empty set and a complete set should be return", func() {
			// TODO: Make setup failures distinct from test
			lineSlice, err := loadLinesFromFile("test_files/all.log", 0)
			if err != nil {
				Panic()
			}
			secondSet := make([]*loglines.ProcessedLine, 0)
			for i, line := range lineSlice.Lines[:300] {
				if i%3 == 0 {
					continue
				}
				secondSet = append(secondSet, line)
			}
			dl1, dl2 := loglines.DedupAB(secondSet, lineSlice.Lines[:300])
			Expect(dl1).To(BeEmpty())
			Expect(dl2).To(Equal(lineSlice.Lines[:300]))
		})
	})
})

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loglines")
}
