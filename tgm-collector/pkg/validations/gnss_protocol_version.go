// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"

	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	GNSSProtID           = TGMEnvVerPath + "/gnss-protocol/"
	gnssProtIDescription = "GNSS protocol version is valid"
	MinProtoVersion      = "29.20"
)

func NewGNSSProtocol(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSVersionFetcher]
	if !ok {
		return nil, fmt.Errorf("gps versions not set in args")
	}

	gnss, ok := rawGPSVer.(*devices.GPSVersions)
	if !ok {
		return nil, fmt.Errorf("could not type cast gps versions")
	}
	v := VersionCheck{
		id:           GNSSProtID,
		Version:      gnss.ProtoVersion,
		checkVersion: gnss.ProtoVersion,
		MinVersion:   MinProtoVersion,
		description:  gnssProtIDescription,
		order:        gnssProtOrdering,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(GNSSProtID, NewGNSSProtocol, []string{datafetcher.GPSVersionFetcher})
}
