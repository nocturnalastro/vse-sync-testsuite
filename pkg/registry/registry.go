// SPDX-License-Identifier: GPL-2.0-or-later

package registry

import (
	"fmt"
)

type BuilderFunc[C any, R any] func(*C) (R, error)

type Registry[B any] struct {
	registry map[string]B
}

func (reg *Registry[B]) register(
	collectorName string,
	builderFunc B,
) {
	reg.registry[collectorName] = builderFunc
}

func (reg *Registry[B]) GetBuilderFunc(collectorName string) (B, error) { //nolint:ireturn // this should be an interface
	builderFunc, ok := reg.registry[collectorName]
	if !ok {
		return builderFunc, fmt.Errorf("not index in registry for collector named %s", collectorName)
	}
	return builderFunc, nil
}

func SetupRegister[B any]() *Registry[B] {
	return &Registry[B]{registry: make(map[string]B)}
}

func Register[B any](registry *Registry[B]) func(string, B) {
	return func(collectorName string, builderFunc B) {
		registry.register(collectorName, builderFunc)
	}
}
