package validations

// SPDX-License-Identifier: GPL-2.0-or-later

import (
	"fmt"
	"strings"

	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	GPSDID          = TGMEnvVerPath + "/gpsd/"
	gpsdDescription = "GPSD Version is valid"
	MinGSPDVersion  = "3.25"
)

func NewGPSDVersion(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSVersionFetcher]
	if !ok {
		return nil, fmt.Errorf("gps versions not set in args")
	}

	gpsdVer, ok := rawGPSVer.(*devices.GPSVersions)
	if !ok {
		return nil, fmt.Errorf("could not type cast gps versions")
	}

	parts := strings.Split(gpsdVer.GPSDVersion, " ")
	v := VersionCheck{
		id:           GPSDID,
		Version:      gpsdVer.GPSDVersion,
		checkVersion: strings.ReplaceAll(parts[0], "~", "-"),
		MinVersion:   MinGSPDVersion,
		description:  gpsdDescription,
		order:        gpsdVersionOrdering,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(GPSDID, NewGPSDVersion, []string{datafetcher.GPSVersionFetcher})
}
