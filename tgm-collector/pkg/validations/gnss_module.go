// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/utils"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations/datafetcher"
)

const (
	expectedModuleName             = "ZED-F9T"
	GNNSSModuleIsCorrect           = TGMEnvModelPath + "/gnss/"
	gnssModuleIsCorrectDescription = "GNSS module is valid"
)

type GNSSModule struct {
	Module string `json:"module"`
}

func (gnssModule *GNSSModule) Verify() error {
	if gnssModule.Module != expectedModuleName {
		return utils.NewInvalidEnvError(
			fmt.Errorf("reported gnss module is not %s", expectedModuleName),
		)
	}
	return nil
}

func (gnssModule *GNSSModule) GetID() string {
	return GNNSSModuleIsCorrect
}

func (gnssModule *GNSSModule) GetDescription() string {
	return gnssModuleIsCorrectDescription
}

func (gnssModule *GNSSModule) GetData() any { //nolint:ireturn // data will vary for each validation
	return gnssModule
}

func (gnssModule *GNSSModule) GetOrder() int {
	return gnssModuleOrdering
}

func NewGNSSModule(args map[string]any) (validationsBase.Validation, error) {
	rawGPSVer, ok := args[datafetcher.GPSVersionFetcher]
	if !ok {
		return nil, fmt.Errorf("gps versions not set in args")
	}

	gpsdVer, ok := rawGPSVer.(*devices.GPSVersions)
	if !ok {
		return nil, fmt.Errorf("could not type cast gps versions")
	}
	return &GNSSModule{Module: gpsdVer.Module}, nil
}

func init() {
	validationsBase.RegisterValidation(GNNSSModuleIsCorrect, NewGNSSModule, []string{datafetcher.GPSVersionFetcher})
}
