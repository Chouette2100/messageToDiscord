package mail

import (
	"strings"
	"testing"
)

func TestParseSimpleMessage(t *testing.T) {
	raw := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: テスト件名\r\n" +
		"Date: Fri, 24 Oct 2026 12:00:00 +0900\r\n" +
		"\r\n" +
		"こんにちは\r\n" +
		"世界\r\n"

	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if msg.From != "sender@example.com" {
		t.Fatalf("From = %q, want %q", msg.From, "sender@example.com")
	}
	if msg.To != "recipient@example.com" {
		t.Fatalf("To = %q, want %q", msg.To, "recipient@example.com")
	}
	if msg.Subject != "テスト件名" {
		t.Fatalf("Subject = %q, want %q", msg.Subject, "テスト件名")
	}
	if msg.Body != "こんにちは\n世界" {
		t.Fatalf("Body = %q, want %q", msg.Body, "こんにちは\n世界")
	}
}

func TestComposeMessageForCron(t *testing.T) {
	raw := ComposeMessage("Cron <root@example.com> test job", "root@example.com", "root@example.com", "command failed\nline 42: permission denied")
	if !contains(raw, "Subject: Cron <root@example.com> test job") {
		t.Fatalf("expected cron-style subject in message, got %q", raw)
	}
	if !contains(raw, "command failed\nline 42: permission denied") {
		t.Fatalf("expected body content in message, got %q", raw)
	}
}

func TestParseMultipartMessage(t *testing.T) {
	boundary := "----=_Part_123"
	raw := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: multipart test\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=\"----=_Part_123\"\r\n\r\n" +
		"This is a multipart message.\r\n" +
		"------=_Part_123\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\n" +
		"aGk=\r\n" +
		"------=_Part_123--\r\n"

	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if msg.Body != "hi" {
		t.Fatalf("Body = %q, want %q", msg.Body, "hi")
	}
	if !contains(msg.Subject, "multipart") {
		t.Fatalf("Subject not parsed correctly: %q", msg.Subject)
	}
	if boundary == "" {
		t.Fatal("invalid test boundary")
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
