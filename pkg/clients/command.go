// SPDX-License-Identifier: GPL-2.0-or-later

package clients

import (
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type Cmder interface {
	GetCommand() (*Command, error)
	ExtractResult(string) (map[string]string, error)
}

type Cmd struct {
	key             string
	prefix          string
	suffix          string
	cmd             string
	outputProcessor func(string) (string, error)
	regex           *regexp.Regexp
	shellRegex      *regexp.Regexp
	fullCmd         string
}

var removeCarrageReturns *regexp.Regexp

func init() {
	removeCarrageReturns = regexp.MustCompile(`\r*\n`)
}
func NewCmd(key, cmd string) (*Cmd, error) {
	cmdInstance := Cmd{
		key:    key,
		cmd:    cmd,
		prefix: fmt.Sprintf("echo '<%s>'", key),
		suffix: fmt.Sprintf("echo '</%s>'", key),
	}

	cmdInstance.fullCmd = fmt.Sprintf("%s;", cmdInstance.prefix)
	cmdInstance.fullCmd += cmdInstance.cmd
	if string(cmd[len(cmd)-1]) != ";" {
		cmdInstance.fullCmd += ";"
	}
	cmdInstance.fullCmd += fmt.Sprintf("%s;", cmdInstance.suffix)

	compiledValueRegex, err := regexp.Compile(`(?s)<` + key + `>\r*\n` + `(.*?)` + `\r*\n</` + key + `>`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex for key %s: %w", key, err)
	}
	cmdInstance.regex = compiledValueRegex
	compiledShellRegex, err := regexp.Compile(`(?s)(<` + key + `>\r*\n.*?\r*\n</` + key + `>)`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile sehll regex for key %s: %w", key, err)
	}
	cmdInstance.shellRegex = compiledShellRegex
	return &cmdInstance, nil
}

func (c *Cmd) SetOutputProcessor(f func(string) (string, error)) {
	c.outputProcessor = f
}

func (c *Cmd) GetCommandString() string {
	return c.fullCmd
}

func (c *Cmd) GetCommand() (*Command, error) {
	cmd := Command{
		stdin: c.GetCommandString(),
		regex: c.shellRegex,
	}
	return &cmd, nil
}

func (c *Cmd) ExtractResult(s string) (map[string]string, error) {
	result := make(map[string]string)
	log.Debugf("extract %s from %s", c.key, s)
	match := c.regex.FindStringSubmatch(s)
	log.Debugf("match %#v", match)

	if len(match) > 0 {
		value := string(removeCarrageReturns.ReplaceAllString(match[1], "\n"))

		if c.outputProcessor != nil {
			cleanValue, err := c.outputProcessor(value)
			if err != nil {
				return result, fmt.Errorf("failed to cleanup value %s of key %s", value, c.key)
			}
			value = cleanValue
		}
		log.Debugf("r %s", value)
		result[c.key] = value
		return result, nil
	}
	return result, fmt.Errorf("failed to find result for key: %s", c.key)
}

type CmdGroup struct {
	cmds []*Cmd
}

func (cgrp *CmdGroup) AddCommand(c *Cmd) {
	cgrp.cmds = append(cgrp.cmds, c)
}

func (cgrp *CmdGroup) GetCommand() (*Command, error) {
	grpCmdStr := ""
	for _, c := range cgrp.cmds {
		grpCmdStr += c.GetCommandString()
	}
	fKey := cgrp.cmds[0].key
	lKey := cgrp.cmds[len(cgrp.cmds)-1].key
	grpRegex, err := regexp.Compile(`(?s:(<` + fKey + `>\r*\n.*\r*\n</` + lKey + `>))`)
	res := &Command{
		stdin: grpCmdStr,
		regex: grpRegex,
	}
	return res, err
}

func (cgrp *CmdGroup) ExtractResult(s string) (map[string]string, error) {
	results := make(map[string]string)
	for _, c := range cgrp.cmds {
		res, err := c.ExtractResult(s)
		if err != nil {
			return results, err
		}
		results[c.key] = res[c.key]
	}
	return results, nil
}
