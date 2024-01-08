// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"time"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

type Collector interface {
	Start() error                                // Setups any internal state required for collection to happen
	Poll(chan PollResult, *utils.WaitGroupCount) // Poll for collectables
	CleanUp() error                              // Stops the collector and cleans up any internal state. It should result in a state that can be started again
	GetPollInterval() time.Duration              // Returns the collectors polling interval
	IsAnnouncer() bool
}

// A union of all values required to be passed into all constructions
type CollectionConstructor struct {
	Callback               callbacks.Callback
	Clientset              *clients.Clientset
	ErroredPolls           chan PollResult
	PTPInterface           string
	Msg                    string
	LogsOutputFile         string
	TempDir                string
	PollInterval           int
	DevInfoAnnouceInterval int
	IncludeLogTimestamps   bool
	KeepDebugFiles         bool
}

type PollResult struct {
	CollectorName string
	Errors        []error
}

type baseCollector struct {
	callback     callbacks.Callback
	ctx          clients.ExecContext
	pollInterval time.Duration
	isAnnouncer  bool
	running      bool
}

func (base *baseCollector) GetPollInterval() time.Duration {
	return base.pollInterval
}

func (base *baseCollector) IsAnnouncer() bool {
	return base.isAnnouncer
}

func (base *baseCollector) Start() error {
	return nil
}

func (base *baseCollector) CleanUp() error {
	base.running = true
	if base.ctx != nil {
		c, ok := base.ctx.(*clients.ReusedConnectionContext)
		if ok {
			c.CloseShell()
		}
	}
	return nil
}

func newBaseCollector(
	pollInterval int,
	isAnnouncer bool,
	callback callbacks.Callback,
	ctx clients.ExecContext,
) *baseCollector {
	return &baseCollector{
		callback:     callback,
		isAnnouncer:  isAnnouncer,
		running:      false,
		pollInterval: time.Duration(pollInterval) * time.Second,
		ctx:          ctx,
	}
}
