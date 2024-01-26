// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import "errors"

//nolint:varnamelen // ok is the standard name for this variable
func getPTPInterface(args map[string]map[string]any) (string, error) {
	ptpArgs, ok := args["PTP"]
	if !ok {
		return "", errors.New("no PTP args in collector args")
	}
	ptpInterfaceRaw, ok := ptpArgs["PtpInterface"]
	if !ok {
		return "", errors.New("no PtpInterface in PTP collector args")
	}

	ptpInterface, ok := ptpInterfaceRaw.(string)
	if !ok {
		return "", errors.New("PTP interface is not a string")
	}
	return ptpInterface, nil
}
