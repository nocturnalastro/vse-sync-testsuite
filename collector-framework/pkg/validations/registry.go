// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"fmt"
	"strings"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
	log "github.com/sirupsen/logrus"
)

type ValidationConstuctor func(map[string]any) (Validation, error)
type dataFetcher func(*clients.Clientset, map[string]any) (any, error)

type entry struct {
	fetcher []string
	builder ValidationConstuctor
}

type ValidationsRegistry struct {
	validations    map[string]entry
	dataCollectors map[string]dataFetcher
}

var registry *ValidationsRegistry

func GetRegistry() *ValidationsRegistry {
	return registry
}

func (reg *ValidationsRegistry) registerDataFunc(
	fetcherName string,
	builderFunc dataFetcher,
) {
	reg.dataCollectors[fetcherName] = builderFunc
}

func (reg *ValidationsRegistry) registerValidation(
	collectorName string,
	builderFunc ValidationConstuctor,
	fetcher []string,
) {
	reg.validations[collectorName] = entry{
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
	log.Info("dc", reg.dataCollectors)
	for _, name := range fetcherNames {
		f, ok := reg.dataCollectors[name]
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

func RegisterValidation(collectorName string, builderFunc ValidationConstuctor, fetcher []string) {
	if registry == nil {
		registry = &ValidationsRegistry{
			validations:    make(map[string]entry, 0),
			dataCollectors: make(map[string]dataFetcher, 0),
		}
	}
	registry.registerValidation(collectorName, builderFunc, fetcher)
}

func RegisterDataFunc(fetcherName string, builderFunc dataFetcher) {
	if registry == nil {
		registry = &ValidationsRegistry{
			validations:    make(map[string]entry, 0),
			dataCollectors: make(map[string]dataFetcher, 0),
		}
	}
	registry.registerDataFunc(fetcherName, builderFunc)
}
