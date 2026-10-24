package discord

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildPayloadIncludesSubjectAndExcerpt(t *testing.T) {
	payload := BuildPayload("送信元@example.com", "テスト件名", "これはとても長い本文です。これはDiscordに転送されたメールです。本文は短く要約されます。")
	if !bytes.Contains(payload, []byte("テスト件名")) {
		t.Fatal("payload does not include subject")
	}
	if !bytes.Contains(payload, []byte("これはとても長い本文です。")) {
		t.Fatal("payload does not include body excerpt")
	}
}

func TestBuildPayloadUsesDiscordSafeUsername(t *testing.T) {
	payload := BuildPayload("sender@example.com", "subject", "body")
	if bytes.Contains(payload, []byte("discord")) {
		t.Fatalf("payload contains forbidden username or text: %s", payload)
	}
}

func TestSendWithRetryRetriesAndSucceeds(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload := []byte(`{"content":"retry me"}`)
	if err := SendWithRetry(server.URL, payload, 4, 0); err != nil {
		t.Fatalf("SendWithRetry returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}
