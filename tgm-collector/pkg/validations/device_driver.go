// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	DeviceDriverVersionID          = TGMEnvVerPath + "/ice-driver/"
	deviceDriverVersionDescription = "Card driver is valid"
)

var (
	minDriverVersion           = "1.11.0"
	minInTreeDriverVersion     = "5.14.0-0"
	outOfTreeIceDriverSegments = 3
)

func NewDeviceDriver(args map[string]any) (validationsBase.Validation, error) {
	rawPTPDevInfo, ok := args[datafetcher.DevInfoFetcher]
	if !ok {
		return nil, fmt.Errorf("dev info not set in args")
	}
	ptpDevInfo, ok := rawPTPDevInfo.(*devices.PTPDeviceInfo)
	if !ok {
		return nil, fmt.Errorf("failed to typecast dev info")
	}

	var err error
	checkVer := ptpDevInfo.DriverVersion
	if checkVer[len(checkVer)-1] == '.' {
		checkVer = checkVer[:len(checkVer)-1]
	}
	ver := fmt.Sprintf("v%s", strings.ReplaceAll(checkVer, "_", "-"))
	if semver.IsValid(ver) {
		if semver.Compare(ver, fmt.Sprintf("v%s", minInTreeDriverVersion)) < 0 {
			err = fmt.Errorf(
				"found device driver version %s. This is below minimum version %s so likely an out of tree driver",
				ptpDevInfo.DriverVersion, minInTreeDriverVersion,
			)
		}
	} else {
		if strings.Count(ptpDevInfo.DriverVersion, ".") == outOfTreeIceDriverSegments {
			err = fmt.Errorf(
				"unable to parse device driver version (%s), likely an out of tree driver",
				ptpDevInfo.DriverVersion,
			)
		}
	}

	v := VersionWithErrorCheck{
		VersionCheck: VersionCheck{
			id:           DeviceDriverVersionID,
			Version:      ptpDevInfo.DriverVersion,
			checkVersion: checkVer,
			MinVersion:   minDriverVersion,
			description:  deviceDriverVersionDescription,
			order:        deviceDriverVersionOrdering,
		},
		Error: err,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(DeviceDriverVersionID, NewDeviceDriver, []string{datafetcher.DevInfoFetcher})
}
