// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors/devices"
)

const (
	gnssProtID           = TGMEnvVerPath + "/gnss-protocol/"
	gnssProtIDescription = "GNSS protocol version is valid"
	MinProtoVersion      = "29.20"
)

func NewGNSSProtocol(gnss *devices.GPSVersions) *VersionCheck {
	return NewVersionCheck(
		gnssProtID,
		gnss.ProtoVersion,
		gnss.ProtoVersion,
		MinProtoVersion,
		gnssProtIDescription,
		gnssProtOrdering,
	)
}
