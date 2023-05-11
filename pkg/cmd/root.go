/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	KubeConfig   string
	PollCount    int
	PollRate     float64
	PTPInterface string

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
	rootCmd.PersistentFlags().StringVar(&KubeConfig, "kubeconfig", "", "kubeconfig file path")
	rootCmd.PersistentFlags().IntVar(&PollCount, "count", 10, "number of queries the cluster (-1) means infinite")
	rootCmd.PersistentFlags().Float64Var(&PollRate, "rate", 1, "poll rate for querying the cluster")
	rootCmd.PersistentFlags().StringVar(&PTPInterface, "interface", "", "ptp interface name")
}
