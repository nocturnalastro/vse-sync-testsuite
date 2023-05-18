// Copyright 2023 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collectors

type CollectedData interface{}

type Collector interface {
	// interfaceName   string
	// ctx             clients.ContainerContext
	// DataTypes       [3]string
	// data            map[string]interface{}
	// inversePollRate float64
	// callback        Callback

	// running  map[string]bool
	// lastPoll time.Time

	Start(key string) error // Links collector to monitoring stack if required
	// Get() (CollectedData, error) // Returns an interface to retrieve data from the monitoring stack
	ShouldPoll() bool           // Check if poll time has alapsed and if it should be polled again
	Poll() []error              // Poll for collectables
	fetchLine() ([]byte, error) // Should call into callback
	CleanUp(key string) error   // Unlinks collecter from monitoring stack if required
}

var Registry map[string]*interface{}

func Register(key string, newCollectorFunc interface{}) {
	Registry[key] = &newCollectorFunc
}

func init() {
	Registry = make(map[string]*interface{})
}
