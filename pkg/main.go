package main

import (
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
	// As this is our only collector then lets crash out
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

func selectCollectorCallback(outputFile string) collectors.Callback {
	if outputFile != "" {
		callback, err := collectors.NewFileCallback(outputFile)
		ifErrorPanic(err)
		return callback
	} else {
		return collectors.StdoutCallBack{}
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
	callback := selectCollectorCallback(cmd.OutputFile)
	ptpCollector, err := collectors.NewPTPCollector(cmd.PTPInterface, ptpContext, cmd.PollRate, callback)
	ifErrorPanic(err)
	err = ptpCollector.Start("all")
	ifErrorPanic(err)
	quit := getQuitChannel()

out:
	for i := 1; cmd.PollCount < 0 || i <= cmd.PollCount; i++ {
		select {
		case <-quit:
			log.Info("ShutingDown")
			ptpCollector.CleanUp("all")
			callback.CleanUp()
			break out
		default:
			if ptpCollector.ShouldPoll() {
				err := ptpCollector.Poll()
				if err != nil {
					// TODO: handle errors (better)
					log.Debug(err)
				}
			}

			time.Sleep(time.Duration(1/cmd.PollRate) * time.Second)
		}

		// If last iteration we should clean up
		if i >= (cmd.PollCount - 1) {
			ptpCollector.CleanUp("all")
		}
	}

	os.Exit(0)
}
