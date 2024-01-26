package validations

// SPDX-License-Identifier: GPL-2.0-or-later

import (
	"strings"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors/devices"
)

const (
	gpsdID          = TGMEnvVerPath + "/gpsd/"
	gpsdDescription = "GPSD Version is valid"
	MinGSPDVersion  = "3.25"
)

func NewGPSDVersion(gpsdVer *devices.GPSVersions) *VersionCheck {
	parts := strings.Split(gpsdVer.GPSDVersion, " ")
	return NewVersionCheck(
		gpsdID,
		gpsdVer.GPSDVersion,
		strings.ReplaceAll(parts[0], "~", "-"),
		MinGSPDVersion,
		gpsdDescription,
		gpsdVersionOrdering,
	)
}
