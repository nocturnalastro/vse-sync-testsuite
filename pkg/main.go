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

package main

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/cmd"
)

func ifErrorPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func setupLogging(logLevel string, out io.Writer) {
	log.SetOutput(out)
	level, err := log.ParseLevel(logLevel)
	ifErrorPanic(err)
	log.SetLevel(level)
}

func main() {
	cmd.Execute()
	setupLogging(cmd.LogLevel, os.Stdout)
}
