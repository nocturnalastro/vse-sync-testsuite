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

package testutils

import (
	"context"
	"fmt"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Return a SPDYExectuor with stdout, stderr and an error embedded
func NewFakeNewSPDYExecutor(
	stdout, stderr string,
	streamErr error,
	execCreationErr error,
) func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeExecutor{method: method, url: url, stdout: stdout, stderr: stderr, err: streamErr}, execCreationErr
	}
}

type fakeExecutor struct {
	err    error
	url    *url.URL
	method string
	stdout string
	stderr string
}

func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	_, err := options.Stdout.Write([]byte(f.stdout))
	if err != nil {
		return fmt.Errorf("failed to write stdout Error: %w", err)
	}
	_, err = options.Stderr.Write([]byte(f.stderr))
	if err != nil {
		return fmt.Errorf("failed to write stderr Error: %w", err)
	}
	return f.err
}

func (f *fakeExecutor) StreamWithContext(ctx context.Context, options remotecommand.StreamOptions) error {
	_, err := options.Stdout.Write([]byte(f.stdout))
	if err != nil {
		return fmt.Errorf("failed to write stdout Error: %w", err)
	}
	_, err = options.Stderr.Write([]byte(f.stderr))
	if err != nil {
		return fmt.Errorf("failed to write stderr Error: %w", err)
	}
	return f.err
}
