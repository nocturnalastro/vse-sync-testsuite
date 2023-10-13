// SPDX-License-Identifier: GPL-2.0-or-later

package collectors //nolint:dupl // new collector

import (
	"fmt"
	"time"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/contexts"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

var (
	GPSCollectorName = "GNSS"
	gpsNavKey        = "gpsNav"
)

type GPSCollector struct {
	callback      callbacks.Callback
	pollInterval  *LockedInterval
	ctx           clients.ContainerContext
	interfaceName string
	running       bool
}

func (gps *GPSCollector) GetPollInterval() time.Duration {
	return gps.pollInterval.interval()
}

func (gps *GPSCollector) ScalePollInterval(factor float64) {
	gps.pollInterval.scale(factor)
}

func (gps *GPSCollector) ResetPollInterval() {
	gps.pollInterval.reset()
}

func (gps *GPSCollector) GetName() string {
	return GPSCollectorName
}

func (gps *GPSCollector) IsAnnouncer() bool {
	return false
}

// Start sets up the collector so it is ready to be polled
func (gps *GPSCollector) Start() error {
	gps.running = true
	return nil
}

func (gps *GPSCollector) poll() error {
	gpsNav, err := devices.GetGPSNav(gps.ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch  %s %w", gpsNavKey, err)
	}
	err = gps.callback.Call(&gpsNav, gpsNavKey)
	if err != nil {
		return fmt.Errorf("callback failed %w", err)
	}
	return nil
}

// Poll collects information from the cluster then
// calls the callback.Call to allow that to persist it
func (gps *GPSCollector) Poll(resultsChan chan PollResult, wg *utils.WaitGroupCount) {
	defer func() {
		wg.Done()
	}()

	errorsToReturn := make([]error, 0)
	err := gps.poll()
	if err != nil {
		errorsToReturn = append(errorsToReturn, err)
	}
	resultsChan <- PollResult{
		CollectorName: GPSCollectorName,
		Errors:        errorsToReturn,
	}
}

// CleanUp stops a running collector
func (gps *GPSCollector) CleanUp() error {
	gps.running = false
	return nil
}

// Returns a new GPSCollector based on values in the CollectionConstructor
func NewGPSCollector(constructor *CollectionConstructor) (Collector, error) {
	ctx, err := contexts.GetPTPDaemonContext(constructor.Clientset)
	if err != nil {
		return &GPSCollector{}, fmt.Errorf("failed to create DPLLCollector: %w", err)
	}

	collector := GPSCollector{
		interfaceName: constructor.PTPInterface,
		ctx:           ctx,
		running:       false,
		callback:      constructor.Callback,
		pollInterval:  NewLockedInterval(constructor.PollInterval),
	}

	return &collector, nil
}

func init() {
	RegisterCollector(GPSCollectorName, NewGPSCollector, optional)
}
