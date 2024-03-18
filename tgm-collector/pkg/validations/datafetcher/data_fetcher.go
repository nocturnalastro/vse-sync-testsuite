package datafetcher

import (
	"log"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/utils"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/contexts"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors/devices"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/common"
)

const (
	DevInfoFetcher    = "devInfo"
	GPSVersionFetcher = "gpsVer"
	GPSNavFetcher     = "gpsNav"
)

func getClientsetFromArgs(args map[string]any) *clients.Clientset {
	rawClientSet, ok := args["clientset"]
	if !ok {
		log.Panic("interfaceName not set in the args")
	}
	clientset, ok := rawClientSet.(*clients.Clientset)
	if !ok {
		log.Panic("could not convert interfaceName in the args to string")
	}
	return clientset
}

//nolint:ireturn // this needs to be an interface
func getDevInfo(args map[string]any) (any, error) {
	clientset := getClientsetFromArgs(args)
	rawCollectorArgs, ok := args["collectorArgs"]
	if !ok {
		log.Panic("collectorArgs not set in the args")
	}
	collectorArgs, ok := rawCollectorArgs.(map[string]map[string]any)
	if !ok {
		log.Panic("could not convert collectorArgs in the args to map[string]map[string]any")
	}
	interfaceName, err := common.GetPTPInterfaceName(collectorArgs)
	utils.IfErrorExitOrPanic(err)

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

func getGPSVersions(args map[string]any) (any, error) {
	clientset := getClientsetFromArgs(args)
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

func getGPSNav(args map[string]any) (any, error) {
	clientset := getClientsetFromArgs(args)

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
