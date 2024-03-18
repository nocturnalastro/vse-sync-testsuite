// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"errors"
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/utils"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	GNSSStatusID          = TGMSyncEnvPath + "/gnss/gpsfix-valid/wpc/"
	gnssStatusDescription = "GNSS Module receiving data"
)

type GNSSNavStatus struct {
	Status *devices.GPSNavStatus `json:"status"`
}

func (gnss *GNSSNavStatus) Verify() error {
	if gnss.Status.GPSFix <= 0 {
		return utils.NewInvalidEnvError(errors.New("GNSS module is not receiving data"))
	}
	return nil
}

func (gnss *GNSSNavStatus) GetID() string {
	return GNSSStatusID
}

func (gnss *GNSSNavStatus) GetDescription() string {
	return gnssStatusDescription
}

func (gnss *GNSSNavStatus) GetData() any { //nolint:ireturn // data will vary for each validation
	return gnss
}

func (gnss *GNSSNavStatus) GetOrder() int {
	return gnssReceivingDataOrdering
}

func NewGNSSNavStatus(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSNavFetcher]
	if !ok {
		return nil, fmt.Errorf("gps versions not set in args")
	}

	gpsDatails, ok := rawGPSVer.(*devices.GPSDetails)
	if !ok {
		return nil, fmt.Errorf("could not type cast gps versions")
	}
	return &GNSSNavStatus{Status: &gpsDatails.NavStatus}, nil
}
