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

package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultCount    int     = 10
	defaultPollRate float64 = 1.0
)

// rootCmd represents the base command when called without any subcommands
var (
	KubeConfig   string
	PollCount    int
	PollRate     float64
	PTPInterface string
	LogLevel     string
	OutputFile   string

	rootCmd = &cobra.Command{
		Use:   "vse-sync-testsuite",
		Short: "A monitoring tool for PTP related metrics",
		Long: `A monitoring tool for PTP related metrics:
			kubeconfig: path to kubeconfig of target system
			count The number of times the cluster will be queried (-1 means infinite)
			rate: The polling rate in seconds
			interface: The interface the PTP configured on
		`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&KubeConfig, "kubeconfig", "k", "", "kubeconfig file path")
	rootCmd.PersistentFlags().IntVarP(
		&PollCount,
		"count",
		"c",
		defaultCount,
		"number of queries the cluster (-1) means infinite",
	)
	rootCmd.PersistentFlags().Float64VarP(
		&PollRate,
		"rate",
		"r",
		defaultPollRate,
		"poll rate for querying the cluster",
	)
	rootCmd.PersistentFlags().StringVarP(&PTPInterface, "interface", "i", "", "ptp interface name")
	rootCmd.PersistentFlags().StringVarP(
		&LogLevel,
		"verbosity",
		"v",
		log.WarnLevel.String(),
		"Log level (debug, info, warn, error, fatal, panic",
	)
	rootCmd.PersistentFlags().StringVarP(&OutputFile, "output", "o", "", "Output file path")
}
