// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"
	"strings"
)

type ValidationConstuctor func(map[string]any) (Validation, error)
type dataFetcher func(map[string]any) (any, error)

type entry struct {
	fetcher []string
	builder ValidationConstuctor
}

type ValidationsRegistry struct {
	validations  map[string]entry
	dataFetchers map[string]dataFetcher
}

var registry *ValidationsRegistry

func GetRegistry() *ValidationsRegistry {
	return registry
}

func (reg *ValidationsRegistry) registerDataFunc(
	fetcherName string,
	fetcherFunc dataFetcher,
) {
	reg.dataFetchers[fetcherName] = fetcherFunc
}

func (reg *ValidationsRegistry) registerValidation(
	validationName string,
	builderFunc ValidationConstuctor,
	fetcher []string,
) {
	reg.validations[validationName] = entry{
		builder: builderFunc,
		fetcher: fetcher,
	}
}

func (reg *ValidationsRegistry) GetBuilderFunc(validationName string) (ValidationConstuctor, []string, error) {
	entry, ok := reg.validations[validationName]
	if !ok {
		return nil, []string{}, fmt.Errorf("no index in validation registry for validation named %s", validationName)
	}
	return entry.builder, entry.fetcher, nil
}

func (reg *ValidationsRegistry) GetDataFetcher(fetcherNames []string) (map[string]dataFetcher, error) {
	missing := make([]string, 0)
	fetcherMap := make(map[string]dataFetcher)
	for _, name := range fetcherNames {
		f, ok := reg.dataFetchers[name]
		if !ok {
			missing = append(missing, name)
		} else {
			fetcherMap[name] = f
		}
	}
	if len(missing) > 0 {
		return fetcherMap, fmt.Errorf("missing fetcher: %s", strings.Join(missing, ","))
	}
	return fetcherMap, nil
}

func createReg() {
	registry = &ValidationsRegistry{
		validations:  make(map[string]entry, 0),
		dataFetchers: make(map[string]dataFetcher, 0),
	}
}

// RegisterValidation
// Note fetched does not need to have been regisitered before the
// validation as this would place implicit constraints on
// import orders as most registractions will happen in init funcs
func RegisterValidation(validationName string, builderFunc ValidationConstuctor, fetcher []string) {
	if registry == nil {
		createReg()
	}
	registry.registerValidation(validationName, builderFunc, fetcher)
}

func RegisterDataFunc(fetcherName string, builderFunc dataFetcher) {
	if registry == nil {
		createReg()
	}
	registry.registerDataFunc(fetcherName, builderFunc)
}
