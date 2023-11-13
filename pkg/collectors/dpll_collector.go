// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"errors"
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/contexts"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

type DPLLCollector struct {
	*baseCollector
	interfaceName string
}

const (
	DPLLCollectorName = "DPLL"
	DPLLInfo          = "dpll-info"
)

// polls for the dpll info then passes it to the callback
func (dpll *DPLLCollector) poll() error {
	dpllInfo, err := devices.GetDevDPLLInfo(dpll.ctx, dpll.interfaceName)

	if err != nil {
		return fmt.Errorf("failed to fetch %s %w", DPLLInfo, err)
	}
	err = dpll.callback.Call(&dpllInfo, DPLLInfo)
	if err != nil {
		return fmt.Errorf("callback failed %w", err)
	}
	return nil
}

// Poll collects information from the cluster then
// calls the callback.Call to allow that to persist it
func (dpll *DPLLCollector) Poll(resultsChan chan PollResult, wg *utils.WaitGroupCount) {
	defer func() {
		wg.Done()
	}()
	errorsToReturn := make([]error, 0)
	err := dpll.poll()
	if err != nil {
		errorsToReturn = append(errorsToReturn, err)
	}
	resultsChan <- PollResult{
		CollectorName: DPLLCollectorName,
		Errors:        errorsToReturn,
	}
}

// Returns a new DPLLCollector from the CollectionConstuctor Factory
func NewDPLLCollector(constructor *CollectionConstructor) (Collector, error) {
	ctx, err := contexts.GetPTPDaemonContext(constructor.Clientset)
	if err != nil {
		return &DPLLCollector{}, fmt.Errorf("failed to create DPLLCollector: %w", err)
	}
	err = devices.BuildDPLLInfoFetcher(constructor.PTPInterface)
	if err != nil {
		return &DPLLCollector{}, fmt.Errorf("failed to build fetcher for DPLLInfo %w", err)
	}

	collector := DPLLCollector{
		baseCollector: newBaseCollectors(
			DPLLCollectorName,
			ctx,
			constructor.PollInterval,
			false,
			constructor.Callback,
		),
		interfaceName: constructor.PTPInterface,
	}

	dpllFSExists, err := devices.IsDPLLFileSystemPresent(collector.ctx, collector.interfaceName)
	if err != nil {
		return &collector, utils.NewRequirementsNotMetError(fmt.Errorf("checking for the DPLL filesystem failed: %w", err))
	}
	if !dpllFSExists {
		return &collector, utils.NewRequirementsNotMetError(errors.New("filesystem with DPLL stats not present"))
	}

	return &collector, nil
}

func init() {
	RegisterCollector(DPLLCollectorName, NewDPLLCollector, optional)
}
