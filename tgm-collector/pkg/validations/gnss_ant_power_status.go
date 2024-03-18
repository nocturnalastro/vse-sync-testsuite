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
	expectedAntStatus        = 2
	GNSSAntStatusID          = TGMSyncEnvPath + "/gnss/antenna-connected/wpc/"
	gnssAntStatusDescription = "GNSS Module is connected to an antenna"
)

type GNSSAntStatus struct {
	Blocks []*devices.GPSAntennaDetails `json:"blocks"`
}

func (gnssAnt *GNSSAntStatus) Verify() error {
	for _, block := range gnssAnt.Blocks {
		if block.Status == expectedAntStatus {
			return nil
		}
	}
	return utils.NewInvalidEnvError(errors.New("no GNSS antenna connected"))
}

func (gnssAnt *GNSSAntStatus) GetID() string {
	return GNSSAntStatusID
}

func (gnssAnt *GNSSAntStatus) GetDescription() string {
	return gnssAntStatusDescription
}

func (gnssAnt *GNSSAntStatus) GetData() any { //nolint:ireturn // data will vary for each validation
	return gnssAnt
}

func (gnssAnt *GNSSAntStatus) GetOrder() int {
	return gnssConnectedToAntOrdering
}

func NewGNSSAntStatus(args map[string]any) (validationsBase.Validation, error) {
	rawGPSNav, ok := args[datafetcher.GPSNavFetcher]
	if !ok {
		return nil, fmt.Errorf("gps nav status not set in args")
	}
	gpsStatus, ok := rawGPSNav.(*devices.GPSDetails)
	if !ok {
		return nil, fmt.Errorf("gps nav status couldn't be type cast")
	}
	return &GNSSAntStatus{Blocks: gpsStatus.AntennaDetails}, nil
}

func init() {
	validationsBase.RegisterValidation(GNSSAntStatusID, NewGNSSAntStatus, []string{datafetcher.GPSNavFetcher})
}
