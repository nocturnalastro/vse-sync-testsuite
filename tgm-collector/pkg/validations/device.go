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
	DeviceDetailsID          = TGMEnvModelPath + "/nic/"
	deviceDetailsDescription = "Card is valid NIC"
)

var (
	VendorIntel        = "0x8086"
	E810WesportChannel = "0x1593"
	E810LoganBeach     = "0x1592"
)

type DeviceDetails struct {
	VendorID string `json:"vendorId"`
	DeviceID string `json:"deviceId"`
}

func (dev *DeviceDetails) Verify() error {
	if dev.VendorID != VendorIntel || (dev.DeviceID != E810WesportChannel && dev.DeviceID != E810LoganBeach) {
		return utils.NewInvalidEnvError(fmt.Errorf("NIC device is not based on E810"))
	}
	return nil
}

func (dev *DeviceDetails) GetID() string {
	return DeviceDetailsID
}

func (dev *DeviceDetails) GetDescription() string {
	return deviceDetailsDescription
}

func (dev *DeviceDetails) GetData() any { //nolint:ireturn // data will very for each validation
	return dev
}

func (dev *DeviceDetails) GetOrder() int {
	return deviceDetailsOrdering
}

func NewDeviceDetails(args map[string]any) (validationsBase.Validation, error) {
	rawPTPDevInfo, ok := args[datafetcher.DevInfoFetcher]
	if !ok {
		return nil, fmt.Errorf("dev info not set in args")
	}
	
	ptpDevInfo, ok := rawPTPDevInfo.(*devices.PTPDeviceInfo)
	if !ok {
		return nil, fmt.Errorf("failed to typecast dev info")
	}

	v := DeviceDetails{
		VendorID: ptpDevInfo.VendorID,
		DeviceID: ptpDevInfo.DeviceID,
	}
	return &v, nil
}

func init() {
	validationsBase.RegisterValidation(DeviceDetailsID, NewDeviceDetails, []string{datafetcher.DevInfoFetcher})
}
