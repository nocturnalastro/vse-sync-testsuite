package common

import "errors"

//nolint:varnamelen // ok is the idomatic name for this var
func GetPTPInterfaceName(collectorArgs map[string]map[string]any) (string, error) {
	ptpArgs, ok := collectorArgs["PTP"]
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
