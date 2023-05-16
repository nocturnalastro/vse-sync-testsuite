package collectors

import (
	"fmt"
	"os"
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
