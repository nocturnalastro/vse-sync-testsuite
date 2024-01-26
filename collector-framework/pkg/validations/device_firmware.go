// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"strings"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors/devices"
)

const (
	deviceFirmwareID          = TGMEnvVerPath + "/nic-firmware/"
	deviceFirmwareDescription = "Card firmware is valid"
)

var (
	MinFirmwareVersion = "4.20"
)

func NewDeviceFirmware(ptpDevInfo *devices.PTPDeviceInfo) *VersionCheck {
	parts := strings.Split(ptpDevInfo.FirmwareVersion, " ")
	return NewVersionCheck(
		deviceFirmwareID,
		ptpDevInfo.FirmwareVersion,
		parts[0],
		MinFirmwareVersion,
		deviceFirmwareDescription,
		deviceFirmwareOrdering,
	)
}
