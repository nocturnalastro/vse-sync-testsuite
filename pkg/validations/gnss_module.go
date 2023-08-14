// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

const (
	expectedModuleName             = "ZED-F9T"
	gnssModuleIsCorrect            = TGMIdBaseURI + "/version/gnss/model/wpc/"
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
	return gnssModuleIsCorrect
}

func (gnssModule *GNSSModule) GetDescription() string {
	return gnssModuleIsCorrectDescription
}

func (gnssModule *GNSSModule) GetData() any { //nolint:ireturn // data will vary for each validation
	return gnssModule
}

func NewGNSSModule(gpsdVer *devices.GPSVersions) *GNSSModule {
	return &GNSSModule{Module: gpsdVer.Module}
}
