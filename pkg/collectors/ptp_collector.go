package collectors

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/collectors/devices"
)

const (
	logFilePermissions = 0666
)

type Callback interface {
	Call(string, string, string) // takes data
	CleanUp()
}

type StdoutCallBack struct {
}

func (c StdoutCallBack) Call(collectorName string, datatype string, line string) {
	fmt.Printf("%v:%v, %v\n", collectorName, datatype, line)
}

func (c StdoutCallBack) CleanUp() {}

func NewFileCallback(filename string) (FileCallBack, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, logFilePermissions)
	if err != nil {
		return FileCallBack{}, err
	}
	return FileCallBack{fileHandle: file}, nil
}

type FileCallBack struct {
	fileHandle *os.File
}

func (c FileCallBack) Call(collectorName string, datatype string, line string) {
	output := fmt.Sprintf("%v:%v, %v\n", collectorName, datatype, line)
	c.fileHandle.Write([]byte(output))
}

func (c FileCallBack) CleanUp() {
	c.fileHandle.Close()
}

type PTPCollector struct {
	interfaceName   string
	ctx             clients.ContainerContext
	DataTypes       [3]string
	data            map[string]interface{}
	inversePollRate float64
	callback        Callback

	running  map[string]bool
	lastPoll time.Time
}

const (
	VendorIntel = "0x8086"
	DeviceE810  = "0x1593"
)

var collectables = [3]string{
	"device-info",
	"dll-info",
	"gnss-tty",
}

func NewPTPCollector(interfaceName string, ctx clients.ContainerContext, pollRate float64, callback Callback) (PTPCollector, error) {
	data := make(map[string]interface{})
	running := make(map[string]bool)

	data["device-info"] = devices.GetPTPDeviceInfo(interfaceName, ctx)
	data["dll-info"] = devices.GetDevDPLLInfo(ctx, interfaceName)

	ptpDevInfo := data["device-info"].(devices.PTPDeviceInfo)
	if ptpDevInfo.VendorID != VendorIntel || ptpDevInfo.DeviceID != DeviceE810 {
		return PTPCollector{}, fmt.Errorf("NIC device is not based on E810")
	}

	collector := PTPCollector{
		interfaceName:   interfaceName,
		ctx:             ctx,
		DataTypes:       collectables,
		data:            data,
		running:         running,
		callback:        callback,
		inversePollRate: 1.0 / float64(pollRate),
		lastPoll:        time.Now(),
	}

	return collector, nil
}

func (ptpDev *PTPCollector) getNotCollectableError(key string) error {
	return fmt.Errorf("key %s is not a colletable of %T", key, ptpDev)
}

func (ptpDev *PTPCollector) getErrorIfNotCollectable(key string) error {
	if _, ok := ptpDev.data[key]; !ok {
		return ptpDev.getNotCollectableError(key)
	} else {
		return nil
	}
}

func (ptpDev PTPCollector) Start(key string) error {
	switch key {
	case "all":
		for _, data_type := range ptpDev.DataTypes[:] {
			log.Debugf("starting: %s", data_type)
			ptpDev.running[data_type] = true
		}
	default:
		err := ptpDev.getErrorIfNotCollectable(key)
		if err != nil {
			return err
		}
		ptpDev.running[key] = true
	}
	return nil
}

// func (ptpDev *PTPCollector) Get(key string) (CollectedData, error) {
// 	value, ok := ptpDev.data[key]
// 	if ok {
// 		return value, nil
// 	}
// 	return nil, fmt.Errorf("key %s is not collectable of %T", key, ptpDev)
// }

// Checks to see if the enou
func (ptpDev PTPCollector) ShouldPoll() bool {
	return time.Since(ptpDev.lastPoll).Seconds() >= ptpDev.inversePollRate
}

func (ptpDev PTPCollector) fetchLine(key string) (line []byte, err error) {
	switch key {
	case "device-info":
		ptpDevInfo := devices.GetPTPDeviceInfo(ptpDev.interfaceName, ptpDev.ctx)
		ptpDev.data["device-info"] = ptpDevInfo
		line, err = json.Marshal(ptpDevInfo)
	case "dll-info":
		dllInfo := devices.GetDevDPLLInfo(ptpDev.ctx, ptpDev.interfaceName)
		ptpDev.data["dll-info"] = dllInfo
		line, err = json.Marshal(dllInfo)
	case "gnss-tty":
		// TODO make lines and timeout configs
		gnssTTYLine := devices.ReadTtyGNSS(ptpDev.ctx, ptpDev.data["device-info"].(devices.PTPDeviceInfo), 1, 1)
		ptpDev.data["gnss-tty"] = gnssTTYLine
		line, err = json.Marshal(gnssTTYLine)
	default:
		return nil, ptpDev.getNotCollectableError(key)
	}
	return line, err
}

// Poll collects infomation from the cluster then
// calls the callback.Call to allow that to persist it
func (ptpDev PTPCollector) Poll() []error {
	errorsToReturn := make([]error, 0)

	for key, is_running := range ptpDev.running {
		if is_running {
			line, err := ptpDev.fetchLine(key)
			// TODO: handle (better)
			if err != nil {
				errorsToReturn = append(errorsToReturn, err)
			}
			ptpDev.callback.Call(fmt.Sprintf("%T", ptpDev), key, string(line))
		}
	}
	return errorsToReturn
}

// Stops a running collector then do clean
func (ptpDev PTPCollector) CleanUp(key string) error {
	switch key {
	case "all":
		ptpDev.running = make(map[string]bool)
	default:
		err := ptpDev.getErrorIfNotCollectable(key)
		if err != nil {
			return err
		}
		delete(ptpDev.running, key)
	}
	return nil
}
