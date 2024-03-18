// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"log"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	"github.com/spf13/cobra"
)

type verifyFunc func(kubeconfig string, useAnalyserJSON bool)

var verify verifyFunc

func SetVerifyFunc(f verifyFunc) {
	verify = f
}

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "environment based actions",
	Long:  `environment based actions`,
}

// VerifyEnvCmd represents the verifyEnv command
var VerifyEnvCmd = &cobra.Command{
	Use:   "verify",
	Short: "verify the environment is ready for collection",
	Long:  `verify the environment is ready for collection`,
}

func runVerify() {
	if verify == nil {
		log.Fatal("Verify command was not registered")
	}
	verify(kubeConfig, useAnalyserJSON)
}

var VerifyEnvCmdOCP = &cobra.Command{
	Use:   "ocp",
	Short: "verify the environment is ready for collection with ocp as a target",
	Long:  `verify the environment is ready for collection with ocp as a target`,
	Run: func(cmd *cobra.Command, args []string) {
		clients.SetRuntimeTarget(clients.TargetOCP)
		runVerify()
	},
}

var VerifyEnvCmdLocal = &cobra.Command{
	Use:   "local",
	Short: "verify the environment is ready for collection with a local target",
	Long:  `verify the environment is ready for collection with a local target`,
	Run: func(cmd *cobra.Command, args []string) {
		clients.SetRuntimeTarget(clients.TargetLocal)
		runVerify()
	},
}

func init() {
	RootCmd.AddCommand(EnvCmd)
	EnvCmd.AddCommand(VerifyEnvCmd)

	VerifyEnvCmd.AddCommand(VerifyEnvCmdOCP)
	AddKubeconfigFlag(VerifyEnvCmdOCP)
	AddOutputFlag(VerifyEnvCmdOCP)
	AddFormatFlag(VerifyEnvCmdOCP)

	VerifyEnvCmd.AddCommand(VerifyEnvCmdLocal)
	AddOutputFlag(VerifyEnvCmdLocal)
	AddFormatFlag(VerifyEnvCmdLocal)

}
