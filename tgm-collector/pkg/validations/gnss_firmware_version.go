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
	GNSSFirmwareID          = TGMEnvVerPath + "/gnss-firmware/"
	gnssDescription = "GNSS Version is valid"
)

var (
	MinGNSSVersion = "2.20"
)

func NewGNSS(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSVersionFetcher]
	if !ok {
		return nil, fmt.Errorf("gps versions not set in args")
	}

	gnss, ok := rawGPSVer.(*devices.GPSVersions)
	if !ok {
		return nil, fmt.Errorf("could not type cast gps versions")
	}

	parts := strings.Split(gnss.FirmwareVersion, " ")
	v := VersionCheck{
		id:           GNSSFirmwareID,
		Version:      gnss.FirmwareVersion,
		checkVersion: parts[1],
		MinVersion:   MinGNSSVersion,
		description:  gnssDescription,
		order:        gnssVersionOrdering,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(GNSSFirmwareID, NewGNSS, []string{datafetcher.GPSVersionFetcher})
}
