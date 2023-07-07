// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/utils"
)

type Params interface {
	CheckForRequiredFields() error
}

func populateParams(cmd *cobra.Command, params Params) error {
	err := viper.Unmarshal(params)
	utils.IfErrorExitOrPanic(err)
	err = params.CheckForRequiredFields()
	if err != nil {
		cmd.PrintErrln(err.Error())
		err = cmd.Usage()
		utils.IfErrorExitOrPanic(err)
		os.Exit(int(utils.InvalidArgs))
		return fmt.Errorf("failed to populate params: %w", err)
	}
	return nil
}
