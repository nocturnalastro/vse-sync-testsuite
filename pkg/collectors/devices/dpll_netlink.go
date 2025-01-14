// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/fetcher"
)

var states = map[string]string{
	"unknown":       "-1",
	"invalid":       "0",
	"freerun":       "1",
	"locked":        "2",
	"locked-ho-acq": "3",
	"holdover":      "4",
}

const (
	OnePPSLabel               = "GNSS-1PPS"
	EEC_OFFSET_PARENT_ID      = 0
	PPS_OFFSET_PARENT_ID      = 1
	DPLL_PHASE_OFFSET_DIVIDER = 1000
)

type DevNetlinkDPLLInfo struct {
	Timestamp string `fetcherKey:"date"       json:"timestamp"`
	EECState  string `fetcherKey:"eec"        json:"eecstate"`
	PPSState  string `fetcherKey:"pps"        json:"state"`
	PPSOffset int64  `fetcherKey:"pps_offset" json:"terror"`
	EECOffset int64  `fetcherKey:"eec_offset" json:"eecterror"`
}

func convertNetlinkOffset(offset int64) float64 {
	// Convert to nano seconds with 3 decimal places
	return float64(int64(math.Round(float64(offset/DPLL_PHASE_OFFSET_DIVIDER)))) / 1000
}

// AnalyserJSON returns the json expected by the analysers
func (dpllInfo *DevNetlinkDPLLInfo) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	formatted := callbacks.AnalyserFormatType{
		ID: "dpll/states",
		Data: map[string]any{
			"timestamp": dpllInfo.Timestamp,
			"eecstate":  dpllInfo.EECState,
			"state":     dpllInfo.PPSState,
			"terror":    convertNetlinkOffset(dpllInfo.PPSOffset),
			"eecterror": convertNetlinkOffset(dpllInfo.EECOffset),
		},
	}
	return []*callbacks.AnalyserFormatType{&formatted}, nil
}

type NetlinkStateEntry struct {
	LockStatus string `json:"lock-status"` //nolint:tagliatelle // not my choice
	Driver     string `json:"module-name"` //nolint:tagliatelle // not my choice
	ClockType  string `json:"type"`        //nolint:tagliatelle // not my choice
	ClockID    uint64 `json:"clock-id"`    //nolint:tagliatelle // not my choice
	ID         int    `json:"id"`          //nolint:tagliatelle // not my choice
}

// # Example output
// [{'clock-id': 5799633565435100136,
//   'id': 0,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'eec'},
//  {'clock-id': 5799633565435100136,
//   'id': 1,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'pps'}]

type NetlinkPin struct {
	Label                string                            `json:"board-label"`         //nolint:tagliatelle // not my choice
	Capabilities         int                               `json:"capabilities"`        //nolint:tagliatelle // not my choice
	ClockID              uint64                            `json:"clock-id"`            //nolint:tagliatelle // not my choice
	Frequency            uint64                            `json:"frequency"`           //nolint:tagliatelle // not my choice
	FrequenciesSupported []*NetlinkFrequencySupportedRange `json:"frequency-supported"` //nolint:tagliatelle // not my choice
	ID                   int32                             `json:"id"`                  //nolint:tagliatelle // not my choice
	ModuleName           string                            `json:"module-name"`         //nolint:tagliatelle // not my choice
	ParentDevices        []*NetlinkParentDevice            `json:"parent-device"`       //nolint:tagliatelle // not my choice
	ParentPins           []*NetlinkParentPin               `json:"parent-pin"`          //nolint:tagliatelle // not my choice
	Type                 string                            `json:"type"`                //nolint:tagliatelle // not my choice
	PhaseAdjust          int32                             `json:"phase-adjust"`        //nolint:tagliatelle // not my choice
	PhaseAdjustMax       int32                             `json:"phase-adjust-max"`    //nolint:tagliatelle // not my choice
	PhaseAdjustMin       int32                             `json:"phase-adjust-min"`    //nolint:tagliatelle // not my choice
}

type NetlinkParentDevice struct {
	Direction   string `json:"direction"`    //nolint:tagliatelle // not my choice
	ParentID    int    `json:"parent-id"`    //nolint:tagliatelle // not my choice
	PhaseOffset int64  `json:"phase-offset"` //nolint:tagliatelle // not my choice
	Prio        int    `json:"prio"`         //nolint:tagliatelle // not my choice
	State       string `json:"state"`        //nolint:tagliatelle // not my choice
}

type NetlinkParentPin struct {
	ParentID int32  `json:"parent-id"` //nolint:tagliatelle // not my choice
	State    string `json:"state"`     //nolint:tagliatelle // not my choice
}

type NetlinkFrequencySupportedRange struct {
	Max int32 `json:"frequency-max"` //nolint:tagliatelle // not my choice
	Min int32 `json:"frequency-min"` //nolint:tagliatelle // not my choice
}

// # Example output
// {
// 	'board-label': 'GNSS-1PPS',
// 	'capabilities': 6,
// 	'clock-id': 5799633565433967608,
// 	'frequency': 1,
// 	'frequency-supported': [
// 		{
// 			'frequency-max': 1,
// 			'frequency-min': 1
// 		}
// 	],
// 	'id': 6,
// 	'module-name': 'ice',
// 	'parent-device': [
// 		{
// 			'direction': 'input',
// 			'parent-id': 0,
// 			'phase-offset': 406616064733390,
// 			'prio': 0,
// 			'state': 'connected'
// 		},
// 		{
// 			'direction': 'input',
// 			'parent-id': 1,
// 			'phase-offset': -1870360,
// 			'prio': 0,
// 			'state': 'connected'
// 		}
// 	],
// 	'phase-adjust': 0,
// 	'phase-adjust-max': 16723,
// 	'phase-adjust-min': -16723,
// 	'type': 'gnss'
// },

var (
	dpllNetlinkFetcher map[uint64]*fetcher.Fetcher
	dpllClockIDFetcher map[string]*fetcher.Fetcher
)

func init() {
	dpllNetlinkFetcher = make(map[uint64]*fetcher.Fetcher)
	dpllClockIDFetcher = make(map[string]*fetcher.Fetcher)
}

func buildPostProcessDPLLNetlink(clockID uint64) fetcher.PostProcessFuncType {
	return func(result map[string]string) (map[string]any, error) {
		processedResult := make(map[string]any)

		entries := make([]NetlinkStateEntry, 0)
		cleaned_device := strings.ReplaceAll(result["dpll-netlink-device"], "'", "\"")
		err := json.Unmarshal([]byte(cleaned_device), &entries)
		if err != nil {
			log.Errorf("Failed to unmarshal netlink device output: %s", err.Error())
		}

		log.Debug("entries: ", entries)
		for _, entry := range entries {
			if entry.ClockID == clockID {
				state, ok := states[entry.LockStatus]
				if !ok {
					log.Errorf("Unknown state: %s", state)
					state = "-1"
				}
				processedResult[entry.ClockType] = state
			}
		}

		pin := NetlinkPin{}
		cleaned_offset_pin := strings.ReplaceAll(result["dpll-netlink-offset"], "'", "\"")
		err = json.Unmarshal([]byte(cleaned_offset_pin), &pin)
		if err != nil {
			log.Errorf("Failed to unmarshal netlink pin output: %s", err.Error())
		}
		for _, parentPin := range pin.ParentDevices {
			switch parentPin.ParentID {
			case EEC_OFFSET_PARENT_ID:
				processedResult["ecc_offset"] = parentPin.PhaseOffset
			case PPS_OFFSET_PARENT_ID:
				processedResult["pps_offset"] = parentPin.PhaseOffset
			}
		}

		return processedResult, nil
	}
}

// BuildDPLLNetlinkDeviceFetcher popluates the fetcher required for
// collecting the DPLLInfo
func BuildDPLLNetlinkDeviceFetcher(params NetlinkParameters) error { //nolint:dupl // Further dedup risks be too abstract or fragile
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key:     "dpll-netlink-device",
				Command: "/linux/tools/net/ynl/cli.py --spec /linux/Documentation/netlink/specs/dpll.yaml --dump device-get",
				Trim:    true,
			},
			{
				Key: "dpll-netlink-offset",
				Command: fmt.Sprintf(
					"/linux/tools/net/ynl/cli.py --spec /linux/Documentation/netlink/specs/dpll.yaml --do pin-get --json %s",
					fmt.Sprintf("'{\"id\": %d}'", params.OffsetPin),
				),
				Trim: true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll netlink: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll netlink: %w", err)
	}
	dpllNetlinkFetcher[params.ClockID] = fetcherInst
	fetcherInst.SetPostProcessor(buildPostProcessDPLLNetlink(params.ClockID))
	return nil
}

// GetDevDPLLInfo returns the device DPLL info for an interface.
func GetDevDPLLNetlinkInfo(ctx clients.ExecContext, params NetlinkParameters) (DevNetlinkDPLLInfo, error) {
	dpllInfo := DevNetlinkDPLLInfo{}
	fetcherInst, fetchedInstanceOk := dpllNetlinkFetcher[params.ClockID]
	if !fetchedInstanceOk {
		err := BuildDPLLNetlinkDeviceFetcher(params)
		if err != nil {
			return dpllInfo, err
		}
		fetcherInst, fetchedInstanceOk = dpllNetlinkFetcher[params.ClockID]
		if !fetchedInstanceOk {
			return dpllInfo, errors.New("failed to create fetcher for DPLLInfo using netlink interface")
		}
	}
	err := fetcherInst.Fetch(ctx, &dpllInfo)
	if err != nil {
		log.Debugf("failed to fetch dpllInfo  via netlink: %s", err.Error())
		return dpllInfo, fmt.Errorf("failed to fetch dpllInfo via netlink: %w", err)
	}
	return dpllInfo, nil
}

func BuildNetlinkInfoFetcher(interfaceName string) error {
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key:     "ignore-this",
				Command: "dnf install -y pciutils",
				Trim:    true,
			},
			{
				Key: "dpll-netlink-clock-id",
				Command: fmt.Sprintf(
					`export IFNAME=%s; export BUSID=$(readlink /sys/class/net/$IFNAME/device | xargs basename | cut -d ':' -f 2,3);`+
						` echo $(("16#$(lspci -v | grep $BUSID -A 20 |grep 'Serial Number' | awk '{print $NF}' | tr -d '-')"))`,
					interfaceName,
				),
				Trim: true,
			},
			{
				Key:     "dpll-netlink-pins",
				Command: "/linux/tools/net/ynl/cli.py --spec /linux/Documentation/netlink/specs/dpll.yaml --dump pin-get",
				Trim:    true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll clock ID: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll clock ID: %w", err)
	}
	fetcherInst.SetPostProcessor(postProcessDPLLNetlinkClockID)
	dpllClockIDFetcher[interfaceName] = fetcherInst
	return nil
}

func postProcessDPLLNetlinkClockID(result map[string]string) (map[string]any, error) {
	processedResult := make(map[string]any)
	clockID, err := strconv.ParseUint(result["dpll-netlink-clock-id"], 10, 64)
	if err != nil {
		return processedResult, fmt.Errorf("failed to parse int for clock id: %w", err)
	}
	processedResult["clockID"] = clockID

	entries := make([]NetlinkPin, 0)
	cleaned_pins := strings.ReplaceAll(result["dpll-netlink-pins"], "'", "\"")
	err = json.Unmarshal([]byte(cleaned_pins), &entries)
	if err != nil {
		log.Errorf("Failed to unmarshal netlink output: %s", err.Error())
	}

	log.Debug("entries: ", entries)
	for _, pin := range entries {
		// TODO: use offset instead maybe?
		if pin.Label == OnePPSLabel && pin.ClockID == clockID {
			processedResult["offsetPin"] = pin.ID
		}
	}

	return processedResult, nil
}

type NetlinkParameters struct {
	Timestamp string `fetcherKey:"date"      json:"timestamp"`
	ClockID   uint64 `fetcherKey:"clockID"   json:"clockId"`
	OffsetPin int32  `fetcherKey:"offsetPin" json:"offsetPin"`
}

func GetNetlinkParameters(ctx clients.ExecContext, interfaceName string) (NetlinkParameters, error) {
	netlinkInfo := NetlinkParameters{}
	fetcherInst, fetchedInstanceOk := dpllClockIDFetcher[interfaceName]
	if !fetchedInstanceOk {
		err := BuildNetlinkInfoFetcher(interfaceName)
		if err != nil {
			return netlinkInfo, err
		}
		fetcherInst, fetchedInstanceOk = dpllClockIDFetcher[interfaceName]
		if !fetchedInstanceOk {
			return netlinkInfo, errors.New("failed to create fetcher for DPLLInfo using netlink interface")
		}
	}
	err := fetcherInst.Fetch(ctx, &netlinkInfo)
	if err != nil {
		log.Debugf("failed to fetch netlink info %s", err.Error())
		return netlinkInfo, fmt.Errorf("failed to fetch netlink info %w", err)
	}
	return netlinkInfo, nil
}
