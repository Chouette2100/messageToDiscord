package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"messageToDiscord/internal/discord"
	"messageToDiscord/internal/mail"
)

/*
sendmailのふりをして受け取ったメールをDiscordに転送する

v.0.1.1 不足していたファイルを追加する。
*/
func main() {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	args := os.Args[1:]
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(args[0], "-") {
		webhookURL = args[0]
		args = args[1:]
	}
	if webhookURL == "" {
		fmt.Fprintln(os.Stderr, "DISCORD_WEBHOOK_URL is required or pass webhook url as the first argument")
		os.Exit(1)
	}

	retries := 3
	if v := os.Getenv("DISCORD_WEBHOOK_RETRIES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			retries = parsed
		}
	}
	backoff := time.Second
	if v := os.Getenv("DISCORD_WEBHOOK_RETRY_DELAY"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			backoff = time.Duration(parsed) * time.Second
		}
	}

	var subject, from, to string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-s", "-sub":
			if i+1 < len(args) {
				subject = args[i+1]
				i++
			}
		case "-f", "-from":
			if i+1 < len(args) {
				from = args[i+1]
				i++
			}
		case "-t":
			// Ignore; recipients are already in the message headers when a full mail is provided.
		case "-i", "-oi":
			// Ignore; already handled by stdin parsing.
		default:
			if !strings.HasPrefix(args[i], "-") {
				if to == "" {
					to = args[i]
				} else {
					to += ", " + args[i]
				}
			}
		}
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read stdin: %v\n", err)
		os.Exit(1)
	}
	if strings.TrimSpace(string(raw)) == "" {
		raw = []byte(mail.ComposeMessage(subject, from, to, ""))
	}
	if strings.Contains(string(raw), "\n\n") == false && (subject != "" || from != "" || to != "") {
		raw = []byte(mail.ComposeMessage(subject, from, to, string(raw)))
	}

	msg, err := mail.Parse(string(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse message: %v\n", err)
		os.Exit(1)
	}

	payload := discord.BuildPayload(msg.From, msg.Subject, msg.Body)
	if err := discord.SendWithRetry(webhookURL, payload, retries, backoff); err != nil {
		failedLog := os.Getenv("DISCORD_FAILED_LOG_PATH")
		if failedLog != "" {
			if writeErr := os.WriteFile(failedLog, payload, 0600); writeErr != nil {
				fmt.Fprintf(os.Stderr, "failed to persist payload to %s: %v\n", failedLog, writeErr)
			}
		}
		fmt.Fprintf(os.Stderr, "failed to send webhook after retries: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("sent: %s\n", msg.Subject)
}
