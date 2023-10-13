// SPDX-License-Identifier: GPL-2.0-or-later

package runner

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors"
)

var resetWait time.Duration = 5 * time.Minute

const (
	preNMEAShortOutage  = "pre_nmea_short_outage"
	postNMEAShortOutage = "nmea_short_outage"
)

type Event struct {
	Data any    `json:"data"`
	ID   string `json:"id"`
}

func (evt Event) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	return []*callbacks.AnalyserFormatType{
		{
			ID:   evt.ID,
			Data: evt.Data,
		},
	}, nil
}

func MonitorStream(reader io.Reader, out callbacks.Callback, instances map[string]collectors.Collector) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		evt := Event{}
		err := json.Unmarshal(line, &evt)
		if err != nil {
			log.Errorf("failed to unmarshal event from stdlin line: %s", line)
		} else {
			handleEvents(evt, instances)
			err = out.Call(evt, "std-in-event")
			if err != nil {
				log.Errorf("failed to output event: %v", evt)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Errorf("Monitoring input failed: %s", err.Error())
	}
}

func handleEvents(evt Event, instances map[string]collectors.Collector) {
	if f, ok := reactions[evt.ID]; ok {
		f(evt, instances)
	}
}

var reactions map[string]func(Event, map[string]collectors.Collector)

func init() {
	reactions = make(map[string]func(Event, map[string]collectors.Collector))
	reactions[preNMEAShortOutage] = handleNMEAOutage
	reactions[postNMEAShortOutage] = handleNMEAOutageReset
}

func doubleRates(instances map[string]collectors.Collector, collectorNames []string) {
	log.Debugf("Double poll rate of collectors: %s", strings.Join(collectorNames, ","))
	for _, name := range collectorNames {
		if c, ok := instances[name]; ok {
			c.ScalePollInterval(0.5) //nolint:gomnd // doubled so times 0.5 not magic
		}
	}
}

func resetRates(instances map[string]collectors.Collector, collectorNames []string) {
	for _, name := range collectorNames {
		if c, ok := instances[name]; ok {
			c.ResetPollInterval()
		}
	}
}

func handleNMEAOutage(evt Event, instances map[string]collectors.Collector) {
	if evt.ID != preNMEAShortOutage {
		return
	}
	collectorNames := []string{
		collectors.DPLLCollectorName,
		collectors.PMCCollectorName,
	}
	doubleRates(instances, collectorNames)
}

func handleNMEAOutageReset(evt Event, instances map[string]collectors.Collector) {
	if evt.ID != postNMEAShortOutage {
		return
	}
	go func() {
		time.Sleep(resetWait)
		collectorNames := []string{
			collectors.DPLLCollectorName,
			collectors.PMCCollectorName,
		}
		resetRates(instances, collectorNames)
	}()
}
