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

import (
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/callbacks"
	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/clients"
)

type Collector interface {
	Start(key string) error   // Setups any internal state required for collection to happen
	ShouldPoll() bool         // Check if poll time has alapsed and if it should be polled again
	Poll() []error            // Poll for collectables
	CleanUp(key string) error // Cleans up any internal state
}

// A union of all values required to be passed into all constuctions
type CollectionConstuctor struct {
	Callback     callbacks.Callback
	Clientset    *clients.Clientset
	PTPInterface string
	Msg          string
	PollRate     float64
}
