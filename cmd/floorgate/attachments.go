package main

import (
	"encoding/base64"
	"fmt"
)

const (
	maxMessageBodyBytes       = 32 << 20
	maxMessageAttachments     = 4
	maxAttachmentBytes        = 8 << 20
	maxMessageAttachmentBytes = 16 << 20
)

type gatewayMessageRequest struct {
	Text        string              `json:"text"`
	Attachments []gatewayAttachment `json:"attachments,omitempty"`
}

// gatewayAttachment deliberately mirrors control.Attachment. Keeping the
// gateway request type local lets this independently deployed boundary validate
// and forward the office wire format without coupling its build to office code.
type gatewayAttachment struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

func validateAttachments(attachments []gatewayAttachment) (int, error) {
	if len(attachments) > maxMessageAttachments {
		return 0, fmt.Errorf("too many attachments")
	}

	total := 0
	for _, attachment := range attachments {
		if !allowedAttachmentMIME(attachment.MimeType) {
			return 0, fmt.Errorf("unsupported attachment mime type")
		}
		decoded, err := base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			return 0, fmt.Errorf("invalid attachment data")
		}
		if len(decoded) > maxAttachmentBytes {
			return 0, fmt.Errorf("attachment too large")
		}
		total += len(decoded)
		if total > maxMessageAttachmentBytes {
			return 0, fmt.Errorf("attachments too large")
		}
	}
	return total, nil
}

func allowedAttachmentMIME(value string) bool {
	switch value {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
