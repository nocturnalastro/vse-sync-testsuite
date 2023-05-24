// SPDX-License-Identifier: GPL-2.0-or-later

package logging

import (
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/utils"
)

func SetupLogging(logLevel string, out io.Writer) {
	log.SetOutput(out)
	level, err := log.ParseLevel(logLevel)
	utils.IfErrorPanic(err)
	log.SetLevel(level)
}
