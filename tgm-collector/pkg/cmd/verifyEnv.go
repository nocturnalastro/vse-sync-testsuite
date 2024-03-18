// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	fCmd "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/cmd"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/verify"
)

func init() {
	AddInterfaceFlag(fCmd.VerifyEnvCmd)
	fCmd.SetVerifyFunc(func(target clients.TargetType, kubeconfig string, useAnalyserJSON bool) {
		verify.Verify(target, ptpInterface, kubeconfig, useAnalyserJSON)
	})
}
