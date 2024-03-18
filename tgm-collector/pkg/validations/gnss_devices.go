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
	HadGNSSDevices            = TGMSyncEnvPath + "/gnss/device-detected/wpc/"
	hadGNSSDevicesDescription = "Has GNSS Devices"
)

type GNSDevices struct {
	Paths []string `json:"paths"`
}

func (gnssDevices *GNSDevices) Verify() error {
	if len(gnssDevices.Paths) == 0 {
		return utils.NewInvalidEnvError(errors.New("no gnss devices found"))
	}
	return nil
}

func (gnssDevices *GNSDevices) GetID() string {
	return HadGNSSDevices
}
func (gnssDevices *GNSDevices) GetDescription() string {
	return hadGNSSDevicesDescription
}

func (gnssDevices *GNSDevices) GetData() any { //nolint:ireturn // data will vary for each validation
	return gnssDevices
}

func (gnssDevices *GNSDevices) GetOrder() int {
	return hasGNSSDevicesOrdering
}

func NewGNSDevices(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSVersionFetcher]
	if !ok {
		return nil, fmt.Errorf("dev info not set in args")
	}
	gpsdVer, ok := rawGPSVer.(*devices.GPSVersions)
	if !ok {
		return nil, fmt.Errorf("cant typecast gps version")
	}
	return &GNSDevices{Paths: gpsdVer.GNSSDevices}, nil
}

func init() {
	validationsBase.RegisterValidation(HadGNSSDevices, NewGNSDevices, []string{datafetcher.GPSVersionFetcher})
}
