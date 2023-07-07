// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/runner"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/utils"
)

var (
	kubeConfig             string
	ptpInterface           string
	outputFile             string
	useAnalyserJSON        bool
	pollCount              int
	pollInterval           int
	devInfoAnnouceInterval int
	collectorNames         []string
)

type CollectionParams struct {
	KubeConfig             string   `mapstructure:"kubeconfig"`
	PTPInterface           string   `mapstructure:"ptp_interface"`
	OutputFile             string   `mapstructure:"output_file"`
	CollectorNames         []string `mapstructure:"collectors"`
	PollCount              int      `mapstructure:"poll_count"`
	PollInterval           int      `mapstructure:"poll_rate"`
	DevInfoAnnouceInterval int      `mapstructure:"announce_rate"`
	UseAnalyserJSON        bool     `mapstructure:"use_analyser_json"`
}

func (p *CollectionParams) CheckForRequiredFields() error {
	missing := make([]string, 0)
	if p.KubeConfig == "" {
		missing = append(missing, "kubeconfig")
	}
	if p.PTPInterface == "" {
		missing = append(missing, "interface")
	}
	if len(missing) > 0 {
		return fmt.Errorf(`required flag(s) "%s" not set`, strings.Join(missing, `", "`))
	}
	return nil
}

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Run the collector tool",
	Long:  `Run the collector tool to gather data from your target cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		runtimeConfig := &CollectionParams{}
		err := populateParams(cmd, runtimeConfig)
		if err == nil {
			collectionRunner := runner.NewCollectorRunner(runtimeConfig.CollectorNames)
			collectionRunner.Run(
				runtimeConfig.KubeConfig,
				runtimeConfig.OutputFile,
				runtimeConfig.PollCount,
				runtimeConfig.PollInterval,
				runtimeConfig.DevInfoAnnouceInterval,
				runtimeConfig.PTPInterface,
				runtimeConfig.UseAnalyserJSON,
			)
		}
	},
}

func init() { //nolint:funlen // Allow this to get a little long
	rootCmd.AddCommand(collectCmd)

	collectCmd.Flags().StringVarP(&kubeConfig, "kubeconfig", "k", "", "Path to the kubeconfig file")
	err := viper.BindPFlag("kubeconfig", collectCmd.Flags().Lookup("kubeconfig"))
	utils.IfErrorExitOrPanic(err)

	collectCmd.Flags().StringVarP(&ptpInterface, "interface", "i", "", "Name of the PTP interface")
	err = viper.BindPFlag("interface", collectCmd.Flags().Lookup("interface"))
	utils.IfErrorExitOrPanic(err)
	viper.RegisterAlias("ptp_interface", "interface")

	collectCmd.Flags().IntVarP(
		&pollCount,
		"count",
		"c",
		defaultCount,
		"Number of queries the cluster (-1) means infinite",
	)
	err = viper.BindPFlag("count", collectCmd.Flags().Lookup("count"))
	utils.IfErrorExitOrPanic(err)
	viper.RegisterAlias("poll_count", "count")

	collectCmd.Flags().IntVarP(
		&pollInterval,
		"rate",
		"r",
		defaultPollInterval,
		"Poll interval for querying the cluster. The value will be polled once ever interval. "+
			"Using --rate 10 will cause the value to be polled once every 10 seconds",
	)
	err = viper.BindPFlag("rate", collectCmd.Flags().Lookup("rate"))
	utils.IfErrorExitOrPanic(err)
	viper.RegisterAlias("poll_rate", "rate")

	collectCmd.Flags().IntVarP(
		&devInfoAnnouceInterval,
		"announce",
		"a",
		defaultDevInfoInterval,
		"interval for announcing the dev info",
	)
	err = viper.BindPFlag("announce", collectCmd.Flags().Lookup("announce"))
	utils.IfErrorExitOrPanic(err)
	viper.RegisterAlias("announce_rate", "announce")

	defaultCollectorNames := make([]string, 0)
	defaultCollectorNames = append(defaultCollectorNames, runner.All)
	collectCmd.Flags().StringSliceVarP(
		&collectorNames,
		"collector",
		"s",
		defaultCollectorNames,
		fmt.Sprintf(
			"the collectors you wish to run (case-insensitive):\n"+
				"\trequired collectors: %s (will be automatically added)\n"+
				"\toptional collectors: %s",
			strings.Join(runner.RequiredCollectorNames, ", "),
			strings.Join(runner.OptionalCollectorNames, ", "),
		),
	)
	err = viper.BindPFlag("collectors", collectCmd.Flags().Lookup("collector"))
	utils.IfErrorExitOrPanic(err)
	viper.RegisterAlias("collector", "collectors")

	collectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to the output file")
	err = viper.BindPFlag("output", collectCmd.Flags().Lookup("output"))
	utils.IfErrorExitOrPanic(err)

	collectCmd.Flags().BoolVarP(
		&useAnalyserJSON,
		"use-analyser-format",
		"j",
		false,
		"Output in a format to be used by analysers from vse-sync-pp",
	)
	err = viper.BindPFlag("use_analyser_format", collectCmd.Flags().Lookup("use-analyser-format"))
	utils.IfErrorExitOrPanic(err)
}
