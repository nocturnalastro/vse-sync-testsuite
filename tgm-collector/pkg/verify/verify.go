// SPDX-License-Identifier: GPL-2.0-or-later

package verify

import (
	"fmt"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/collectors"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/runner"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/utils"
	validationsBase "github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/validations"
	"github.com/redhat-partner-solutions/vse-sync-collection-tools/tgm-collector/pkg/validations"
)

const (
	unknownMsgPrefix = "The following error occurred when trying to gather environment data for the following validations"
	antPowerRetries  = 3
)

func reportAnalyserJSON(results []*ValidationResult) {
	callback, err := callbacks.SetupCallback("-", callbacks.AnalyserJSON)
	utils.IfErrorExitOrPanic(err)

	sort.Slice(results, func(i, j int) bool {
		return results[i].validation.GetOrder() < results[j].validation.GetOrder()
	})

	anyHasFailed := false
	for _, res := range results {
		if res.resType == resTypeFailure {
			anyHasFailed = true
		}
		err := callback.Call(res, "env-check")
		if err != nil {
			log.Errorf("callback failed during validation %s", err.Error())
		}
	}

	if anyHasFailed {
		os.Exit(int(utils.InvalidEnv))
	}
}

//nolint:funlen,cyclop // allow slightly long function
func report(results []*ValidationResult, useAnalyserJSON bool) {
	if useAnalyserJSON {
		reportAnalyserJSON(results)
		return
	}

	failures := make([]*ValidationResult, 0)
	unknown := make([]*ValidationResult, 0)

	for _, res := range results {
		//nolint:exhaustive // Not reporting successes so no need to gather them
		switch res.resType {
		case resTypeFailure:
			failures = append(failures, res)
		case resTypeUnknown:
			unknown = append(unknown, res)
		}
	}

	// Report unknowns along side failures
	if len(unknown) > 0 {
		dataErrors := make([]error, 0)
		for _, res := range unknown {
			dataErrors = append(dataErrors, res.GetPrefixedError())
		}
		log.Error(utils.MakeCompositeError(unknownMsgPrefix, dataErrors))
	}

	switch {
	case len(failures) > 0:
		validationsErrors := make([]error, 0)
		for _, res := range failures {
			validationsErrors = append(validationsErrors, res.GetPrefixedError())
		}
		err := utils.MakeCompositeInvalidEnvError(validationsErrors)
		utils.IfErrorExitOrPanic(err)
	case len(unknown) > 0:
		// If only unknowns print this message
		fmt.Println("Some checks did not complete, it is likely something is not correct in the environment") //nolint:forbidigo // This to print out to the user
	default:
		fmt.Println("No issues found.") //nolint:forbidigo // This to print out to the user
	}
}

func getValidations(interfaceName string, kubeConfig string) []validationsBase.Validation {
	collectorNamers := runner.GetCollectorsToRun(clients.GetRuntimeTarget(), []string{runner.All})
	validationsNames := make([]string, 0)
	validationsNames = append(validationsNames, validations.GeneralValidations()...)

	collectorRegistry := collectors.GetRegistry()
	for _, cName := range collectorNamers {
		validationsNames = append(validationsNames, collectorRegistry.GetValidations(cName)...)
	}

	validationsRegistry := validationsBase.GetRegistry()

	datafetchers, err := validationsRegistry.GetDataFetcher(validationsNames)
	utils.IfErrorExitOrPanic(err)

	clientset, err := clients.GetClientset(kubeConfig)
	utils.IfErrorExitOrPanic(err)

	args := map[string]any{
		"clientset":     clientset,
		"interfaceName": interfaceName,
	}

	for key, df := range datafetchers {
		args[key], err = df(clientset, args)
		utils.IfErrorExitOrPanic(err)
	}

	validations := make([]validationsBase.Validation, 0)
	for _, vName := range validationsNames {
		f, _, _ := validationsRegistry.GetBuilderFunc(vName)
		v, err := f(args)
		utils.IfErrorExitOrPanic(err)
		validations = append(validations, v)
	}
	return validations
}

func Verify(interfaceName, kubeConfig string, useAnalyserJSON bool) {
	checks := getValidations(interfaceName, kubeConfig)
	results := make([]*ValidationResult, 0)
	for _, check := range checks {
		results = append(results, NewValidationResult(check))
	}

	report(results, useAnalyserJSON)
}
