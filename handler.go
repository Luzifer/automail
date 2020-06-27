package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/jhillyerd/enmime"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type mailHandler struct {
	Match   []matcher `yaml:"match"`
	Command []string  `yaml:"command"`
}

func (m mailHandler) Handles(msg *enmime.Envelope) bool {
	for _, ma := range m.Match {
		if ma.Match(msg) {
			return true
		}
	}
	return false
}

func (m mailHandler) Process(imapClient *client.Client, msg *imap.Message, envelope *enmime.Envelope) error {
	cmd := exec.Command(m.Command[0], m.Command[1:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Unable to create stdin pipe")
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "Unable to create stdout pipe")
	}
	defer stdout.Close()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if scanner.Err() != nil {
				log.WithError(err).Error("Stdout errored")
				return
			}

			var cw = new(commandTypeWrap)
			if err := json.Unmarshal(scanner.Bytes(), cw); err != nil {
				log.WithError(err).Error("Unable to unmarshal command")
				continue
			}

			log.WithFields(log.Fields{
				"type": cw.Type,
				"uid":  msg.Uid,
			}).Debug("Received command from script")

			c, err := cw.rewrap(scanner.Bytes())
			if err != nil {
				log.WithError(err).Error("Unable to parse command")
				continue
			}

			if err = c.Execute(imapClient, msg, envelope, stdin); err != nil {
				log.WithError(err).Error("Unable to execute command")
				continue
			}
		}
	}()

	if err = cmd.Start(); err != nil {
		return errors.Wrap(err, "Unable to start process")
	}

	if err = json.NewEncoder(stdin).Encode(mailToTransport(envelope)); err != nil {
		return errors.Wrap(err, "Unable to send mail to process")
	}

	return errors.Wrap(cmd.Wait(), "Process exited unclean")
}

type matcher struct {
	Any      bool    `yaml:"any"`
	Header   string  `yaml:"header"`
	Exact    *string `yaml:"exact"`
	Includes *string `yaml:"includes"`
	RegExp   *string `yaml:"regexp"`
}

func (m matcher) Match(msg *enmime.Envelope) bool {
	if m.Any {
		return true
	}

	return m.matchString(msg.GetHeader(strings.ToLower(m.Header)))
}

func (m matcher) matchString(s string) bool {
	if m.Exact != nil && s == *m.Exact {
		return true
	}

	if m.Includes != nil && strings.Contains(s, *m.Includes) {
		return true
	}

	if m.RegExp != nil && regexp.MustCompile(*m.RegExp).MatchString(s) {
		return true
	}

	return false
}
