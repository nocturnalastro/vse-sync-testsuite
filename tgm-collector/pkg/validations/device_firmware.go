// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"
	"strings"

	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	DeviceFirmwareID          = TGMEnvVerPath + "/nic-firmware/"
	deviceFirmwareDescription = "Card firmware is valid"
)

var (
	MinFirmwareVersion = "4.20"
)

func NewDeviceFirmware(args map[string]any) (validationsBase.Validation, error) {
	rawPTPDevInfo, ok := args[datafetcher.DevInfoFetcher]
	if !ok {
		return nil, fmt.Errorf("dev info not set in args")
	}
	ptpDevInfo, ok := rawPTPDevInfo.(*devices.PTPDeviceInfo)
	if !ok {
		return nil, fmt.Errorf("failed to typecast dev info")
	}

	parts := strings.Split(ptpDevInfo.FirmwareVersion, " ")
	v := VersionCheck{
		id:           DeviceFirmwareID,
		Version:      ptpDevInfo.FirmwareVersion,
		checkVersion: parts[0],
		MinVersion:   MinFirmwareVersion,
		description:  deviceFirmwareDescription,
		order:        deviceFirmwareOrdering,
	}

	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(DeviceFirmwareID, NewDeviceFirmware, []string{datafetcher.DevInfoFetcher})
}
