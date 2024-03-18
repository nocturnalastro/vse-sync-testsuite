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
	log.Info(collectorNamers)
	for _, cName := range collectorNamers {
		validationsNames = append(validationsNames, collectorRegistry.GetValidations(cName)...)
	}

	validationsRegistry := validationsBase.GetRegistry()
	log.Info(validationsNames)
	validationFuncs := make([]validationsBase.ValidationConstuctor, 0)
	fetcherNames := make([]string, 0)
	for _, vName := range validationsNames {
		builderFunc, fetchers, err := validationsRegistry.GetBuilderFunc(vName)
		utils.IfErrorExitOrPanic(err)
		validationFuncs = append(validationFuncs, builderFunc)
		fetcherNames = append(fetcherNames, fetchers...)
	}

	datafetchers, err := validationsRegistry.GetDataFetcher(fetcherNames)
	utils.IfErrorExitOrPanic(err)

	clientset, err := clients.GetClientset(kubeConfig)
	utils.IfErrorExitOrPanic(err)

	args := map[string]any{
		"clientset":     clientset,
		"interfaceName": interfaceName,
	}
	log.Info(datafetchers)
	for key, df := range datafetchers {
		log.Info(key)
		args[key], err = df(clientset, args)
		utils.IfErrorExitOrPanic(err)
	}
	log.Info("args", args)
	validations := make([]validationsBase.Validation, 0)
	for _, f := range validationFuncs {
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
