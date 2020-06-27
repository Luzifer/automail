package main

import (
	"encoding/json"
	"io"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
)

type command interface {
	Execute(*client.Client, *imap.Message, io.Writer) error
}

type commandTypeWrap struct {
	Type string `json:"type"`
}

func (c commandTypeWrap) rewrap(data []byte) (command, error) {
	var out command

	switch c.Type {

	case "move":
		out = new(commandMove)

	case "add_flags":
		out = new(commandAddFlags)

	case "del_flags":
		out = new(commandDelFlags)

	case "set_flags":
		out = new(commandSetFlags)

	default:
		return nil, errors.New("Command not found")

	}

	return out, errors.Wrap(json.Unmarshal(data, out), "Unable to unmarshal into command")
}

type commandMove struct {
	ToMailbox string `json:"to_mailbox"`
}

func (c commandMove) Execute(client *client.Client, msg *imap.Message, stdin io.Writer) error {
	s := &imap.SeqSet{}
	s.AddNum(msg.Uid)

	if err := client.UidCopy(s, c.ToMailbox); err != nil {
		return errors.Wrap(err, "Unable to copy to target mailbox")
	}

	return errors.Wrap(
		client.UidStore(s, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.DeletedFlag}, nil),
		"Unable to set deleted flag in original mailbox",
	)
}

type commandAddFlags struct {
	Flags []string `json:"flags"`
}

func (c commandAddFlags) Execute(client *client.Client, msg *imap.Message, stdin io.Writer) error {
	var (
		flags []interface{}
		s     = &imap.SeqSet{}
	)
	s.AddNum(msg.Uid)

	for _, f := range c.Flags {
		flags = append(flags, f)
	}

	return errors.Wrap(
		client.UidStore(s, imap.FormatFlagsOp(imap.AddFlags, true), flags, nil),
		"Unable to add flags",
	)
}

type commandDelFlags struct {
	Flags []string `json:"flags"`
}

func (c commandDelFlags) Execute(client *client.Client, msg *imap.Message, stdin io.Writer) error {
	var (
		flags []interface{}
		s     = &imap.SeqSet{}
	)
	s.AddNum(msg.Uid)

	for _, f := range c.Flags {
		flags = append(flags, f)
	}

	return errors.Wrap(
		client.UidStore(s, imap.FormatFlagsOp(imap.RemoveFlags, true), flags, nil),
		"Unable to remove flags",
	)
}

type commandSetFlags struct {
	Flags []string `json:"flags"`
}

func (c commandSetFlags) Execute(client *client.Client, msg *imap.Message, stdin io.Writer) error {
	var (
		flags []interface{}
		s     = &imap.SeqSet{}
	)
	s.AddNum(msg.Uid)

	for _, f := range c.Flags {
		flags = append(flags, f)
	}

	return errors.Wrap(
		client.UidStore(s, imap.FormatFlagsOp(imap.SetFlags, true), flags, nil),
		"Unable to set flags",
	)
}
