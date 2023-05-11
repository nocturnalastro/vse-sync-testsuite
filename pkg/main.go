// Copyright 2023 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/cmd"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/collectors"
)

// TODO make config
var (
	PTPNamespace  string = "openshift-ptp"
	PodNamePrefix string = "linuxptp-daemon-"
	PTPContainer  string = "linuxptp-daemon-container"
)

func ifErrorPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func getQuitChannel() chan os.Signal {
	// Allow ourselves to handle shut down gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

func setupLogging(logLevel string, out io.Writer) {
	log.SetOutput(out)
	level, err := log.ParseLevel(logLevel)
	ifErrorPanic(err)
	log.SetLevel(level)
}

//nolint:ireturn // The point of this function is to return a callback but which one is dependant on the input
func selectCollectorCallback(outputFile string) (collectors.Callback, error) {
	if outputFile != "" {
		callback, err := collectors.NewFileCallback(outputFile)
		return callback, fmt.Errorf("failed to create callback %w", err)
	} else {
		return collectors.StdoutCallBack{}, nil
	}
}

func main() {
	cmd.Execute()
	setupLogging(cmd.LogLevel, os.Stdout)

	log.Debugf("Kubeconfig: %s\n", cmd.KubeConfig)
	log.Debugf("PollRate: %v\n", cmd.PollRate)
	log.Debugf("PTPInterface: %s\n", cmd.PTPInterface)

	clientset := clients.GetClientset(cmd.KubeConfig)
	ptpContext, err := clients.NewContainerContext(clientset, PTPNamespace, PodNamePrefix, PTPContainer)
	ifErrorPanic(err)
	callback, err := selectCollectorCallback(cmd.OutputFile)
	ifErrorPanic(err)
	ptpCollector, err := collectors.NewPTPCollector(cmd.PTPInterface, ptpContext, cmd.PollRate, callback)
	ifErrorPanic(err)
	err = ptpCollector.Start(collectors.All)
	ifErrorPanic(err)
	quit := getQuitChannel()

out:
	for numberOfPolls := 1; cmd.PollCount < 0 || numberOfPolls <= cmd.PollCount; numberOfPolls++ {
		select {
		case <-quit:
			log.Info("Killed shuting down")
			break out
		default:
			if ptpCollector.ShouldPoll() {
				errors := ptpCollector.Poll()
				if len(errors) > 0 {
					// TODO: handle errors (better)
					log.Error(errors)
				}
			}
			time.Sleep(time.Duration(1/cmd.PollRate) * time.Second)
		}
	}

	errColletor := ptpCollector.CleanUp(collectors.All)
	errCallback := callback.CleanUp()
	ifErrorPanic(errColletor)
	ifErrorPanic(errCallback)

	os.Exit(0)
}
