// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"errors"
	"log"
	"os"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	fCmd "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/cmd"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/runner"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/utils"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/collectors"
	"github.com/spf13/cobra"
)

const (
	defaultIncludeLogTimestamps bool   = false
	defaultTempDir              string = "."
	defaultKeepDebugFiles       bool   = false
	tempdirPerm                        = 0755
)

type CollectorArgFunc func() map[string]map[string]any

var (
	logsOutputFile       string
	includeLogTimestamps bool
	tempDir              string
	keepDebugFiles       bool
)

func setCommonFlags(cmd *cobra.Command) {
	AddInterfaceFlag(cmd)
	cmd.Flags().StringVarP(
		&logsOutputFile,
		"logs-output", "l", "",
		"Path to the logs output file. This is required when using the logs collector",
	)
	cmd.Flags().BoolVar(
		&includeLogTimestamps,
		"log-timestamps", defaultIncludeLogTimestamps,
		"Specifies if collected logs should include timestamps or not. (default is false)",
	)
	cmd.Flags().StringVarP(&tempDir, "tempdir", "t", defaultTempDir,
		"Directory for storing temp/debug files. Must exist.")
	cmd.Flags().BoolVar(&keepDebugFiles, "keep", defaultKeepDebugFiles, "Keep debug files")
}

func init() { //nolint:funlen // Allow this to get a little long
	setCommonFlags(fCmd.CollectOCP)
	setCommonFlags(fCmd.CollectLocal)

	fCmd.SetCollecterArgsFunc(func(selectedCollectors []string) map[string]map[string]any {
		// Check args
		collectorNames := runner.GetCollectorsToRun(clients.GetRuntimeTarget(), selectedCollectors)
		for _, c := range collectorNames {
			if (c == collectors.LogsCollectorName || c == runner.All) && logsOutputFile == "" {
				utils.IfErrorExitOrPanic(utils.NewMissingInputError(
					errors.New("if Logs collector is selected you must also provide a log output file")),
				)
			}
		}

		tempDir, err := utils.ExpandUser(tempDir)
		if err != nil {
			log.Fatal(err)
		}

		if err := os.MkdirAll(tempDir, tempdirPerm); err != nil {
			log.Fatal(err)
		}

		// Populate collector args
		collectorArgs := make(map[string]map[string]any)
		collectorArgs["PTP"] = map[string]any{
			"ptpInterface": ptpInterface,
		}
		collectorArgs["Logs"] = map[string]any{
			"logsOutputFile":       logsOutputFile,
			"includeLogTimestamps": includeLogTimestamps,
			"tempDir":              tempDir,
			"keepDebugFiles":       keepDebugFiles,
		}
		return collectorArgs
	})
}
