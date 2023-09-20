// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"

	"golang.org/x/mod/semver"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

const (
	deviceDriverVersionID          = TGMEnvVerPath + "/ice-driver/"
	deviceDriverVersionDescription = "Card driver is valid"
)

var (
	minDriverVersion       = "1.11.0"
	minInTreeDriverVersion = "5.14.0"
)

type DeviceDriverCheck struct {
	MinOOTVersion string
	VersionCheck
}

func (dc *DeviceDriverCheck) Verify() error {
	ver := fmt.Sprintf("v%s", dc.Version)
	if !semver.IsValid(ver) {
		return fmt.Errorf("unable to parse device driver version (%s)", dc.Version)
	}
	if semver.Compare(ver, fmt.Sprintf("v%s", dc.MinVersion)) < 0 {
		return utils.NewInvalidEnvError(fmt.Errorf("unexpected version: %s < %s", dc.Version, dc.MinVersion))
	}
	return nil
}

func (dc *DeviceDriverCheck) IsLikelyOOT() bool {
	ver := fmt.Sprintf("v%s", dc.Version)

	if semver.Compare(ver, fmt.Sprintf("v%s", dc.MinOOTVersion)) >= 0 &&
		semver.Major(ver) == "1" {
		return true
	}
	return false
}

func NewDeviceDriver(ptpDevInfo *devices.PTPDeviceInfo) *DeviceDriverCheck {
	return &DeviceDriverCheck{
		MinOOTVersion: minDriverVersion,
		VersionCheck: VersionCheck{
			id:           deviceDriverVersionID,
			Version:      ptpDevInfo.DriverVersion,
			checkVersion: ptpDevInfo.DriverVersion,
			MinVersion:   minInTreeDriverVersion,
			description:  deviceDriverVersionDescription,
			order:        deviceDriverVersionOrdering,
		},
	}
}
