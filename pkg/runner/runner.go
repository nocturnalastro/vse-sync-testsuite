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

package runner

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/collectors"
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

//nolint:ireturn // The point of this function is to return a callback but which one is dependant on the input
func selectCollectorCallback(outputFile string) (callbacks.Callback, error) {
	if outputFile != "" {
		callback, err := callbacks.NewFileCallback(outputFile)
		return callback, fmt.Errorf("failed to create callback %w", err)
	} else {
		return callbacks.StdoutCallBack{}, nil
	}
}

var (
	collectorsToInit = [1]string{"PTP"}
)

func setupLogging(logLevel string, out io.Writer) {
	log.SetOutput(out)
	level, err := log.ParseLevel(logLevel)
	ifErrorPanic(err)
	log.SetLevel(level)
}
func setupCollectors(
	callback callbacks.Callback,
	ptpInterface string,
	clientset *clients.Clientset,
	pollRate float64,
) []collectors.Collector {
	collecterInstances := make([]collectors.Collector, 0)
	var newCollector collectors.Collector

	constuctor := collectors.CollectionConstuctor{
		Callback:     callback,
		PTPInterface: ptpInterface,
		Clientset:    clientset,
		PollRate:     pollRate,
	}

	for _, constuctorName := range collectorsToInit {
		switch constuctorName {
		case "PTP":
			NewPTPCollector, err := constuctor.NewPTPCollector() //nolint:govet // TODO clean this up
			ifErrorPanic(err)
			newCollector, _ = NewPTPCollector.(collectors.Collector)
		default:
			panic("Unknown collector")
		}
		if newCollector != nil {
			collecterInstances = append(collecterInstances, newCollector)
		}
	}
	return collecterInstances
}

func Run(
	kubeConfig string,
	logLevel string,
	outputFile string,
	pollCount int,
	pollRate float64,
	ptpInterface string,
) {
	setupLogging(logLevel, os.Stdout)

	clientset := clients.GetClientset(kubeConfig)
	callback, err := selectCollectorCallback(outputFile)
	ifErrorPanic(err)

	collecterInstances := setupCollectors(callback, ptpInterface, clientset, pollRate)

	for _, collector := range collecterInstances {
		err = collector.Start(collectors.All)
		ifErrorPanic(err)
	}

	quit := getQuitChannel()

out:
	for numberOfPolls := 1; pollCount < 0 || numberOfPolls <= pollCount; numberOfPolls++ {
		select {
		case <-quit:
			log.Info("Killed shuting down")
			break out
		default:
			for _, collector := range collecterInstances {
				if collector.ShouldPoll() {
					errors := collector.Poll()
					if len(errors) > 0 {
						// TODO: handle errors (better)
						log.Error(errors)
					}
				}
			}
			time.Sleep(time.Duration(1/pollRate) * time.Second)
		}
	}
	for _, collector := range collecterInstances {
		errColletor := collector.CleanUp(collectors.All)
		ifErrorPanic(errColletor)
	}
	errCallback := callback.CleanUp()
	ifErrorPanic(errCallback)
}
