package main

import (
	"encoding/base64"

	"github.com/jhillyerd/enmime"
)

type attachmentTransport struct {
	FileName    string
	Content     string
	ContentType string
}

func attachmentFromMail(msg *enmime.Envelope, filename string) *attachmentTransport {
	for _, a := range msg.Attachments {
		if a.FileName == filename {
			return &attachmentTransport{
				Content:     base64.StdEncoding.EncodeToString(a.Content),
				ContentType: a.ContentType,
				FileName:    a.FileName,
			}
		}
	}

	return nil
}

type mailTransport struct {
	Attachments []string          `json:"attachments"`
	Headers     map[string]string `json:"headers"`
	HTML        string            `json:"html"`
	Text        string            `json:"text"`
}

func mailToTransport(msg *enmime.Envelope) *mailTransport {
	var out = &mailTransport{
		Headers: map[string]string{},
		HTML:    msg.HTML,
		Text:    msg.Text,
	}

	for _, a := range msg.Attachments {
		out.Attachments = append(out.Attachments, a.FileName)
	}

	for _, hn := range msg.GetHeaderKeys() {
		out.Headers[hn] = msg.GetHeader(hn)
	}

	return out
}
