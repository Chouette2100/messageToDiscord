package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"messageToDiscord/internal/discord"
	"messageToDiscord/internal/mail"

	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/exsrapi/v2"
)

/*
sendmailのふりをして受け取ったメールをDiscordに転送する

v.0.1.1 不足していたファイルを追加する。
v.0.1.2 Webhook URLを設定ファイルから読み込み、メッセージを（テスト用に）ファイルから読み込めるようにする。
        logファイルを作成する。
*/
const version = "000102"

func readRawInput(args []string) ([]byte, error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-F", "--file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing path after %s", args[i])
			}
			path := args[i+1]
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read input file %q: %w", path, err)
			}
			return data, nil
		}
	}
	return io.ReadAll(os.Stdin)
}

func main() {

	        // ログファイルの作成
        logfile, err := exsrapi.CreateLogfile(version, exsrapi.Version, srapi.Version)
        if err != nil {
                log.Printf("ログファイルの作成に失敗しました。%v\n", err)
                return
        }
        defer logfile.Close()

        // フォアグラウンド（端末に接続されているか）を判定
        isForeground := term.IsTerminal(int(os.Stdout.Fd()))
        if isForeground {
                // フォアグラウンドならログファイル + コンソール
                log.SetOutput(io.MultiWriter(logfile, os.Stdout))
        } else {
                // バックグラウンドならログファイルのみ
                log.SetOutput(logfile)
        }

        log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
        log.Printf("Version=%s Start\n", version)

	type Config struct {
		WebhookURL string `yaml:"webhook_url"`
		Retries    int    `yaml:"retries"`
		RetryDelay int    `yaml:"retry_delay"`
	}
	var config = &Config{}
	err = exsrapi.LoadConfig("config.enc.yml", config)
	if err != nil {
		log.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	args := os.Args[1:]
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(args[0], "-") {
		webhookURL = args[0]
		args = args[1:]
	}
	if webhookURL == "" {
		log.Printf("DISCORD_WEBHOOK_URL is required or pass webhook url as the first argument")
		// os.Exit(1)
		if config.WebhookURL != "" {
			webhookURL = config.WebhookURL
		} else {
			log.Printf("Webhook URL is not set in config file either.")
			os.Exit(1)
		}
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
		case "-F", "--file":
			if i+1 < len(args) {
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

	raw, err := readRawInput(args)
	if err != nil {
		log.Printf("failed to read stdin/input file: %v\n", err)
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
		log.Printf("failed to parse message: %v\n", err)
		os.Exit(1)
	}

	payload := discord.BuildPayload(msg.From, msg.Subject, msg.Body)
	if err := discord.SendWithRetry(webhookURL, payload, retries, backoff); err != nil {
		failedLog := os.Getenv("DISCORD_FAILED_LOG_PATH")
		if failedLog != "" {
			if writeErr := os.WriteFile(failedLog, payload, 0600); writeErr != nil {
				log.Printf("failed to persist payload to %s: %v\n", failedLog, writeErr)
			}
		}
		log.Printf("failed to send webhook after retries: %v\n", err)
		os.Exit(1)
	}

	log.Printf("sent: %s\n", msg.Subject)
}
