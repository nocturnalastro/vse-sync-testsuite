// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"strings"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors/devices"
)

const (
	gnssID          = TGMEnvVerPath + "/gnss-firmware/"
	gnssDescription = "GNSS Version is valid"
)

var (
	MinGNSSVersion = "2.20"
)

func NewGNSS(gnss *devices.GPSVersions) *VersionCheck {
	parts := strings.Split(gnss.FirmwareVersion, " ")
	return NewVersionCheck(
		gnssID,
		gnss.FirmwareVersion,
		parts[1],
		MinGNSSVersion,
		gnssDescription,
		gnssVersionOrdering,
	)
}
