package collectors

import (
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/collectors/devices"
)

type Collectables struct {
	key       string
	container interface{}
}

type PTPCollector struct {
	interfaceName string
	ctx           clients.ContainerContext
	DataTypes     [4]string
	data          map[string]interface{}
}

var collectables = [4]string{
	"device-info",
	"dll-info",
	"logs",
}

func NewPTPCollector(interfaceName string, ctx clients.ContainerContext, logFilename string) PTPCollector {
	data := make(map[string]interface{})
	data["device-info"] = devices.GetPTPDeviceInfo(interfaceName, ctx)
	data["dll-info"] = devices.GetDevDPLLInfo(ctx, interfaceName)
	data["logs"] = devices.NewPTPLogsInterface(ctx, logFilename)
	return PTPCollector{
		interfaceName: interfaceName,
		ctx:           ctx,
		DataTypes:     collectables,
		data:          data,
	}
}

func (ptpDev *PTPCollector) Start(key string) error {
	if key == "logs" || key == "all" {
		return ptpDev.data["logs"].(devices.PTPLogsInterface).Start()
	}
	return nil

}
func (ptpDev *PTPCollector) Get(key string) interface{} {
	return ptpDev.data[key]
}

func (ptpDev *PTPCollector) CleanUp() {
	ptpDev.data["logs"].(devices.PTPLogsInterface).CleanUp()
}
