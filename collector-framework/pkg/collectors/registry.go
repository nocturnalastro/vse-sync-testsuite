// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"fmt"
	"log"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/collector-framework/pkg/clients"
)

type collectonBuilderFunc func(*CollectionConstructor) (Collector, error)
type collectorInclusionType int

type runTarget int
type entry struct {
	name        string
	target      runTarget
	validations []string
}

const (
	Required collectorInclusionType = iota
	Optional

	RunOnAll runTarget = iota
	RunOnOCP
	RunOnLocal
)

type CollectorRegistry struct {
	registry map[string]collectonBuilderFunc
	all      map[string]*entry
	required []*entry
	optional []*entry
}

var registry *CollectorRegistry

func GetRegistry() *CollectorRegistry {
	return registry
}

func (reg *CollectorRegistry) register(
	collectorName string,
	builderFunc collectonBuilderFunc,
	inclusionType collectorInclusionType,
	target runTarget,
	validations []string,
) {
	reg.registry[collectorName] = builderFunc
	entry := &entry{name: collectorName, target: target, validations: validations}
	switch inclusionType {
	case Required:
		reg.required = append(reg.required, entry)
	case Optional:
		reg.optional = append(reg.optional, entry)
	default:
		log.Panic("Incorrect collector inclusion type")
	}
	reg.all[collectorName] = entry
}

func (reg *CollectorRegistry) GetBuilderFunc(collectorName string) (collectonBuilderFunc, error) {
	builderFunc, ok := reg.registry[collectorName]
	if !ok {
		return nil, fmt.Errorf("not index in registry for collector named %s", collectorName)
	}
	return builderFunc, nil
}

func fromTargetTypeToRunOn(target clients.TargetType) runTarget {
	switch target {
	case clients.TargetOCP:
		return RunOnOCP
	case clients.TargetLocal:
		return RunOnLocal
	}
	return RunOnAll
}

func getFilteredTargets(entries []*entry, target clients.TargetType) []string {
	names := make([]string, 0)
	runOn := fromTargetTypeToRunOn(target)
	for _, entry := range entries {
		if entry.target == RunOnAll || entry.target == runOn {
			names = append(names, entry.name)
		}
	}
	return names
}

func (reg *CollectorRegistry) GetRequiredNames(target clients.TargetType) []string {
	return getFilteredTargets(reg.required, target)
}

func (reg *CollectorRegistry) GetOptionalNames(target clients.TargetType) []string {
	return getFilteredTargets(reg.optional, target)
}

func (reg *CollectorRegistry) GetValidations(collectorName string) []string {
	return reg.all[collectorName].validations
}

func RegisterCollector(
	collectorName string,
	builderFunc collectonBuilderFunc,
	inclusionType collectorInclusionType,
	target runTarget,
	validations []string,
) {
	if registry == nil {
		registry = &CollectorRegistry{
			registry: make(map[string]collectonBuilderFunc, 0),
			all:      make(map[string]*entry),
			required: make([]*entry, 0),
			optional: make([]*entry, 0),
		}
	}
	registry.register(collectorName, builderFunc, inclusionType, target, validations)
}
