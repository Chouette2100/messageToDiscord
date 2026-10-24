# cron 連携の例

## 基本例

```cron
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
DISCORD_WEBHOOK_RETRIES=3
DISCORD_WEBHOOK_RETRY_DELAY=2
DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json

0 * * * * /usr/local/bin/backup.sh 2>&1 | /usr/local/bin/sendmail-to-discord
```

## この例での意味

- `backup.sh` の標準出力とエラー出力をそのままパイプで渡す
- `sendmail-to-discord` が本プログラムのラッパーとして動く
- `stderr` や `stdout` の内容がメール本文として扱われる
- Discord にエラー内容が通知される

## 実行ラッパー例

```bash
#!/bin/bash
/usr/local/bin/go run /home/chouette/go/src/messageToDiscord/. "$@"
```

もしくは、ビルド済みバイナリを使う場合:

```bash
#!/bin/bash
/usr/local/bin/message-to-discord -s "Cron <$(hostname)> job failed" -f "root@$(hostname)"
```

---

## 注意点

- cron は環境変数を通常自動的に引き継がないため、`crontab` 内で明示的に設定する
- バックアップやジョブの失敗時は `2>&1` を付けて stderr を含める
- 失敗しても通知されないケースがあるため、ログ確認が必要
