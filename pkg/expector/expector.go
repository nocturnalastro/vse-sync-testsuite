// SPDX-License-Identifier: GPL-2.0-or-later

package expecter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	expect "github.com/Netflix/go-expect"
	log "github.com/sirupsen/logrus"

	"github.com/redhat-partner-solutions/vse-sync-collection-tools/pkg/clients"
)

// sh-4.4#
var promptRE = regexp.MustCompile(`(sh-\d.\d#\s*)`) // prompt is full of carriage returns every time except the first time.

// OcpExpecter manages the expecter and debug shell for an OCP debug shell.
type OcpExpecter struct {
	expecter *expect.Console
	outLog   strings.Builder
}

func (exp *OcpExpecter) GetOutLog() string {
	return exp.outLog.String()
}

// WaitFor waits for a single item from expectOpts to be matched.
func (exp *OcpExpecter) WaitFor(expectOpts ...expect.ExpectOpt) (string, error) {
	buf, err := exp.expecter.Expect(expectOpts...)
	if err != nil {
		log.Debugf(`failed to find match; got buffer %q; got err %s`, buf, err.Error())
		return "", fmt.Errorf("failed to find match: %w", err)
	}
	log.Debugf(`found match; buffer: %q`, buf)
	exp.outLog.WriteString(buf)
	return buf, nil
}

func (exp *OcpExpecter) WaitForPrompt() error {
	_, err := exp.WaitFor(expect.Regexp(promptRE))
	return err
}

func (exp *OcpExpecter) RunCommand(command string) error {
	log.Debugf(`running command: "%s"`, command)
	n, err := exp.expecter.SendLine(command)
	if err != nil {
		if n > 0 {
			e := fmt.Errorf("sent incomplete line: '%s', encountered error '%w'", command[:n], err)
			log.Error(e)
			return e
		}
		e := fmt.Errorf("could not run command: '%s', encountered error '%w'", command, err)
		log.Error(e)
		return e
	} else {
		return nil
	}
}

func (exp *OcpExpecter) RunCommandAndWaitFor(command string, expectOpts ...expect.ExpectOpt) (string, error) {
	err := exp.RunCommand(command)
	if err != nil {
		return "", err
	}
	return exp.WaitFor(expectOpts...)
}

func (exp *OcpExpecter) RunCommandAndWaitForPrompt(command string) (string, error) {
	err := exp.RunCommand(command)
	if err != nil {
		return "", err
	}
	return exp.WaitFor(expect.Regexp(promptRE))
}

func (exp *OcpExpecter) Close() {
	err := exp.RunCommand("quit")
	if err != nil {
		log.Errorf("failed when closing shell: %s", err.Error())
	}
	err = exp.expecter.Close()
	if err != nil {
		log.Infof("failed to close expector: %s", err.Error())
	}
}

// Spawn an expecter with a connection to the ptp daemon container.  Return an expecter and command in a state ready for the first input.
func NewPTPDaemonDebugExpecter(ctx clients.ContainerContext, timeout time.Duration) (*OcpExpecter, error) {
	expecter, err := expect.NewConsole(expect.WithDefaultTimeout(timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to start console: %w", err)
	}

	ctx.OpenShell(expecter.Tty())
	exp := OcpExpecter{
		expecter: expecter,
	}
	return &exp, nil
}
