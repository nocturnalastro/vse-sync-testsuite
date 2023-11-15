// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/utils"
)

const (
	TGMTestIDBase   = "https://github.com/redhat-partner-solutions/vse-sync-test/tree/main/tests"
	TGMEnvModelPath = TGMTestIDBase + "/environment/model"
	TGMEnvVerPath   = TGMTestIDBase + "/environment/version"
	TGMSyncEnvPath  = TGMTestIDBase + "/sync/G.8272/environment/status"
)

const (
	clusterVersionOrdering int = iota
	ptpOperatorVersionOrdering
	gpsdVersionOrdering
	deviceDetailsOrdering
	deviceDriverVersionOrdering
	deviceFirmwareOrdering
	gnssModuleOrdering
	gnssVersionOrdering
	gnssProtOrdering
	hasGNSSDevicesOrdering
	gnssConnectedToAntOrdering
	gnssReceivingDataOrdering
	configuredForGrandMasterOrdering
)

type VersionCheck struct {
	id           string `json:"-"`
	Version      string `json:"version"`
	Expected     string `json:"expected"`
	checkVersion string `json:"-"`
	minVersion   string `json:"-"`
	exactVersion string `json:"-"`
	description  string `json:"-"`
	order        int    `json:"-"`
}

func (verCheck *VersionCheck) setExpected() {
	if verCheck.exactVersion != "" {
		verCheck.Expected = verCheck.exactVersion
	} else {
		verCheck.Expected = verCheck.minVersion
	}
}

func (verCheck *VersionCheck) verifyMinVersion() error {
	ver := fmt.Sprintf("v%s", strings.ReplaceAll(verCheck.checkVersion, "_", "-"))
	if !semver.IsValid(ver) {
		return fmt.Errorf("could not parse version %s", ver)
	}
	if semver.Compare(ver, fmt.Sprintf("v%s", verCheck.minVersion)) < 0 {
		return utils.NewInvalidEnvError(
			fmt.Errorf("unexpected version: %s < %s", verCheck.checkVersion, verCheck.minVersion),
		)
	}
	return nil
}

func (verCheck *VersionCheck) verifyRequiredVersion() error {
	if verCheck.Version != verCheck.exactVersion {
		return fmt.Errorf("unexpected version: %s != %s", verCheck.Version, verCheck.exactVersion)
	}
	return nil
}

func (verCheck *VersionCheck) Verify() error {
	if verCheck.exactVersion != "" {
		return verCheck.verifyRequiredVersion()
	} else {
		return verCheck.verifyMinVersion()
	}
}

func (verCheck *VersionCheck) GetID() string {
	return verCheck.id
}

func (verCheck *VersionCheck) GetDescription() string {
	return verCheck.description
}

func (verCheck *VersionCheck) GetData() any { //nolint:ireturn // data will vary for each validation
	verCheck.setExpected()
	return verCheck
}

func (verCheck *VersionCheck) GetOrder() int {
	return verCheck.order
}

type VersionWithError struct {
	Error   error  `json:"fetchError"`
	Version string `json:"version"`
}

func MarshalVersionAndError(ver *VersionWithError) ([]byte, error) {
	var err any
	if ver.Error != nil {
		err = ver.Error.Error()
	}
	marsh, marshalErr := json.Marshal(&struct {
		Error   any    `json:"fetchError"`
		Version string `json:"version"`
	}{
		Version: ver.Version,
		Error:   err,
	})
	return marsh, fmt.Errorf("failed to marshal VersionWithError %w", marshalErr)
}

type VersionWithErrorCheck struct {
	Error error
	VersionCheck
}

func (verCheck *VersionWithErrorCheck) MarshalJSON() ([]byte, error) {
	return MarshalVersionAndError(&VersionWithError{
		Version: verCheck.Version,
		Error:   verCheck.Error,
	})
}

func (verCheck *VersionWithErrorCheck) Verify() error {
	if verCheck.Error != nil {
		return verCheck.Error
	}
	return verCheck.VersionCheck.Verify()
}

type ExactCheckValues struct {
	ClusterVersion        string
	DeviceDriverVersion   string
	DeviceFirmwareVersion string
	// DeviceID       string
	// AntStatus      string
	// GNSSDevice     string
	GNSSFirmwareVersion string
	GNSSModule          string
	GNSSProtocol        string
	// GPSFix       string
	GPSDVersion string
	// GMFlag string
	OperatorVersion string
}
