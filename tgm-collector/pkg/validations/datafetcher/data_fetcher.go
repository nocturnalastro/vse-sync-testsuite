package datafetcher

import (
	"log"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/contexts"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
)

const (
	DevInfoFetcher    = "devInfo"
	GPSVersionFetcher = "gpsVer"
	GPSNavFetcher     = "gpsNav"
)

//nolint:ireturn // this needs to be an interface
func getDevInfo(clientset *clients.Clientset, args map[string]any) (any, error) {
	rawInterfaceName, ok := args["interfaceName"]
	if !ok {
		log.Panic("interfaceName not set in the args")
	}
	interfaceName, ok := rawInterfaceName.(string)
	if !ok {
		log.Panic("could not convert interfaceName in the args to string")
	}
	ctx, err := contexts.GetPTPDaemonContextOrLocal(clientset)
	if err != nil {
		return nil, err
	}
	devInfo, err := devices.GetPTPDeviceInfo(interfaceName, ctx)
	if err != nil {
		return nil, err
	}
	return &devInfo, nil
}

func getGPSVersions(clientset *clients.Clientset, args map[string]any) (any, error) {
	ctx, err := contexts.GetPTPDaemonContextOrLocal(clientset)
	if err != nil {
		return nil, err
	}
	gnssVersions, err := devices.GetGPSVersions(ctx)
	if err != nil {
		return nil, err
	}
	return &gnssVersions, nil
}

func getGPSNav(clientset *clients.Clientset, args map[string]any) (any, error) {
	ctx, err := contexts.GetPTPDaemonContextOrLocal(clientset)
	if err != nil {
		return nil, err
	}
	gpsDetails, err := devices.GetGPSNav(ctx)
	if err != nil {
		return nil, err
	}
	return &gpsDetails, nil
}

func init() {
	validationsBase.RegisterDataFunc(DevInfoFetcher, getDevInfo)
	validationsBase.RegisterDataFunc(GPSVersionFetcher, getGPSVersions)
	validationsBase.RegisterDataFunc(GPSNavFetcher, getGPSNav)
}
