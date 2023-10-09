// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/fetcher"
)

const (
	unitConversionFactor = 100
)

type DevDPLLInfo struct {
	Timestamp string  `fetcherKey:"date"          json:"timestamp"`
	EECState  string  `fetcherKey:"dpll_0_state"  json:"eecstate"`
	PPSState  string  `fetcherKey:"dpll_1_state"  json:"state"`
	PPSOffset float64 `fetcherKey:"dpll_1_offset" json:"terror"`
}

// AnalyserJSON returns the json expected by the analysers
func (dpllInfo *DevDPLLInfo) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	formatted := callbacks.AnalyserFormatType{
		ID: "dpll/time-error",
		Data: map[string]any{
			"timestamp": dpllInfo.Timestamp,
			"eecstate":  dpllInfo.EECState,
			"state":     dpllInfo.PPSState,
			"terror":    dpllInfo.PPSOffset / unitConversionFactor,
		},
	}
	return []*callbacks.AnalyserFormatType{&formatted}, nil
}

var (
	dpllFetcher map[string]*fetcher.Fetcher
)

func init() {
	dpllFetcher = make(map[string]*fetcher.Fetcher)
}

func postProcessDPLL(result map[string]string) (map[string]any, error) {
	processedResult := make(map[string]any)
	offset, err := strconv.ParseFloat(result["dpll_1_offset"], 32)
	if err != nil {
		return processedResult, fmt.Errorf("failed converting dpll_1_offset %w to an int", err)
	}
	processedResult["dpll_1_offset"] = offset
	return processedResult, nil
}

// BuildDPLLInfoFetcher popluates the fetcher required for
// collecting the DPLLInfo
func BuildDPLLInfoFetcher(interfaceName string) error { //nolint:dupl // Further dedup risks be too abstract or fragile
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key:     "dpll_0_state",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_0_state", interfaceName),
				Trim:    true,
			},
			{
				Key:     "dpll_1_state",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_1_state", interfaceName),
				Trim:    true,
			},
			{
				Key:     "dpll_1_offset",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_1_offset", interfaceName),
				Trim:    true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll: %w", err)
	}
	dpllFetcher[interfaceName] = fetcherInst
	fetcherInst.SetPostProcessor(postProcessDPLL)
	return nil
}

func getDevDPLLInfoFetcher(ctx clients.ContainerContext, interfaceName string) (*fetcher.Fetcher, error) {
	fetcherInst, fetchedInstanceOk := dpllFetcher[interfaceName]
	if !fetchedInstanceOk {
		err := BuildDPLLInfoFetcher(interfaceName)
		if err != nil {
			return fetcherInst, err
		}
		fetcherInst, fetchedInstanceOk = dpllFetcher[interfaceName]
		if !fetchedInstanceOk {
			return fetcherInst, errors.New("failed to create fetcher for DPLLInfo")
		}
	}
	if !fetcherInst.IsRunning() {
		err := fetcherInst.Start(ctx)
		if err != nil {
			return fetcherInst, fmt.Errorf("failed to start fetcher for DPLLInfo: %w", err)
		}
	}
	return fetcherInst, nil
}

// GetDevDPLLInfo returns the device DPLL info for an interface.
func GetDevDPLLInfo(ctx clients.ContainerContext, interfaceName string) (DevDPLLInfo, error) {
	dpllInfo := DevDPLLInfo{}
	fetcherInst, err := getDevDPLLInfoFetcher(ctx, interfaceName)
	if err != nil {
		log.Debugf("failed to find dpllInfo fetcher %s", err.Error())
		return dpllInfo, fmt.Errorf("failed to find dpllInfo fetcher %w", err)
	}
	err = fetcherInst.Fetch(ctx, &dpllInfo)
	if err != nil {
		log.Debugf("failed to fetch dpllInfo %s", err.Error())
		return dpllInfo, fmt.Errorf("failed to fetch dpllInfo %w", err)
	}
	return dpllInfo, nil
}

func IsDPLLFileSystemPresent(ctx clients.ContainerContext, interfaceName string) (bool, error) {
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{},
		[]fetcher.AddCommandArgs{
			{
				Key:     "paths",
				Command: fmt.Sprintf("ls -1 /sys/class/net/%s/device/", interfaceName),
				Trim:    true,
			},
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to build fetcher to check DPLL FS  %w", err)
	}
	type Paths struct {
		Paths string `fetcherKey:"paths"`
	}
	paths := Paths{}
	expected := map[string]bool{
		"dpll_0_state":  false,
		"dpll_1_state":  false,
		"dpll_1_offset": false,
	}
	err = fetcherInst.Start(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to start fetcher for check: %w", err)
	}
	err = fetcherInst.Fetch(ctx, &paths)
	if err != nil {
		return false, fmt.Errorf("failed to check DPLL FS  %w", err)
	}
	for _, p := range strings.Split(paths.Paths, "\n") {
		for expectedPath := range expected {
			if strings.Trim(p, " ") == expectedPath {
				expected[expectedPath] = true
			}
		}
	}
	for _, value := range expected {
		if !value {
			return false, nil
		}
	}
	return true, nil
}

func CloseDPLLInfoFetcher(interfaceName string) {
	fetcherInst, fetchedInstanceOk := dpllFetcher[interfaceName]
	if fetchedInstanceOk {
		fetcherInst.Close()
	}
}
