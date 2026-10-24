package mail

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// Message represents a parsed mail message.
type Message struct {
	From    string
	To      string
	Subject string
	Date    string
	Body    string
}

// Parse parses a raw MIME email from a string.
func Parse(raw string) (Message, error) {
	if strings.TrimSpace(raw) == "" {
		return Message{}, fmt.Errorf("empty message")
	}

	headers, body, err := splitHeadersAndBody(raw)
	if err != nil {
		return Message{}, err
	}

	msgBody := decodeBody(body, headers)
	msg := Message{
		From:    normalizeAddress(headers.Get("From")),
		To:      normalizeAddress(headers.Get("To")),
		Subject: decodeHeader(headers.Get("Subject")),
		Date:    decodeHeader(headers.Get("Date")),
		Body:    normalizeBody(msgBody),
	}

	if msg.From == "" {
		msg.From = "unknown"
	}
	if msg.Subject == "" {
		msg.Subject = "(no subject)"
	}

	return msg, nil
}

func splitHeadersAndBody(raw string) (textproto.MIMEHeader, string, error) {
	full := strings.ReplaceAll(raw, "\r\n", "\n")
	reader := textproto.NewReader(bufio.NewReader(strings.NewReader(full)))
	headers, err := reader.ReadMIMEHeader()
	if err != nil {
		return nil, "", err
	}

	parts := strings.SplitN(full, "\n\n", 2)
	if len(parts) < 2 {
		return headers, "", nil
	}
	return headers, parts[1], nil
}

func normalizeAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if addr, err := mail.ParseAddress(value); err == nil {
		return addr.Address
	}
	return value
}

func decodeHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoder := new(mime.WordDecoder)
	decoded, err := decoder.DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}

func normalizeBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	body = strings.TrimRight(body, "\n")
	return body
}

func decodeBody(raw string, headers textproto.MIMEHeader) string {
	contentType := headers.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			if decoded, ok := decodeMultipartBody(raw, boundary); ok {
				return decoded
			}
		}
	}

	transferEncoding := strings.ToLower(strings.TrimSpace(headers.Get("Content-Transfer-Encoding")))
	return decodeTransferEncoding(raw, transferEncoding)
}

func decodeMultipartBody(raw, boundary string) (string, bool) {
	reader := multipart.NewReader(strings.NewReader(raw), boundary)
	var parts []string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false
		}

		bodyBytes, err := io.ReadAll(part)
		if err != nil {
			continue
		}
		partBody := decodeTransferEncoding(string(bodyBytes), strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Transfer-Encoding"))))
		if part.Header.Get("Content-Type") == "" || strings.HasPrefix(strings.ToLower(part.Header.Get("Content-Type")), "text/") {
			parts = append(parts, partBody)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, "\n"), true
}

func decodeTransferEncoding(raw, transferEncoding string) string {
	switch transferEncoding {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
		if err != nil {
			if rawDecoded, rawErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(raw)); rawErr == nil {
				return string(rawDecoded)
			}
			return raw
		}
		return string(decoded)
	case "quoted-printable":
		b, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(raw)))
		if err != nil {
			return raw
		}
		return string(b)
	default:
		return raw
	}
}

// ComposeMessage creates a minimal RFC-822 style email from cron/sendmail-like inputs.
func ComposeMessage(subject, from, to, body string) string {
	if subject == "" {
		subject = "(no subject)"
	}
	if from == "" {
		from = "cron@localhost"
	}
	if to == "" {
		to = "root@localhost"
	}
	date := time.Now().Format(time.RFC1123Z)
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	body = strings.TrimRight(body, "\n")

	return "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n" +
		"Date: " + date + "\n" +
		"MIME-Version: 1.0\n" +
		"Content-Type: text/plain; charset=UTF-8\n\n" +
		body + "\n"
}
