// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	fCmd "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/cmd"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/verify"
)

func init() {
	AddInterfaceFlag(fCmd.VerifyEnvCmd)
	fCmd.SetVerifyFunc(func(kubeconfig string, useAnalyserJSON bool) {
		verify.Verify(ptpInterface, kubeconfig, useAnalyserJSON)
	})
}
