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
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/callbacks"
)

type AnouncementCollector struct {
	callback callbacks.Callback
	key      string
	msg      string
}

func (anouncer *AnouncementCollector) Start(key string) error {
	anouncer.key = key
	return nil
}

func (anouncer *AnouncementCollector) ShouldPoll() bool {
	// We always want to annouce ourselves
	return true
}

func (anouncer *AnouncementCollector) Poll() []error {
	errs := make([]error, 1)
	err := anouncer.callback.Call(
		fmt.Sprintf("%T", anouncer),
		anouncer.key,
		anouncer.msg,
	)
	if err != nil {
		errs[0] = err
	}
	return errs
}

func (anouncer *AnouncementCollector) CleanUp(key string) error {
	return nil
}

func (constuctor *CollectionConstuctor) NewAnouncementCollector() (*AnouncementCollector, error) {
	anouncer := AnouncementCollector{callback: constuctor.Callback, msg: constuctor.Msg}
	return &anouncer, nil
}
