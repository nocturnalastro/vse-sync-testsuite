// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"errors"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	collectorsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/contexts"
)

//nolint:varnamelen // ok is the idomatic name for this var
func getPTPInterfaceName(constructor *collectorsBase.CollectionConstructor) (string, error) {
	ptpArgs, ok := constructor.CollectorArgs["PTP"]
	if !ok {
		return "", errors.New("no PTP args in collector args")
	}
	ptpInterfaceRaw, ok := ptpArgs["ptpInterface"]
	if !ok {
		return "", errors.New("no ptpInterface in PTP collector args")
	}

	ptpInterface, ok := ptpInterfaceRaw.(string)
	if !ok {
		return "", errors.New("PTP interface is not a string")
	}
	return ptpInterface, nil
}

// Empty function used to insure the collectors module gets added to the resulting binary
func IncludeCollectorsNoOp() {}

func getPTPDaemonContext(c *clients.Clientset) (clients.ExecContext, error) {
	return contexts.GetPTPDaemonContext(c)
}
