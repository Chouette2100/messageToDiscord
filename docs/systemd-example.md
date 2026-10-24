# systemd の導入例

## 1. バイナリをビルドする

```bash
cd /home/chouette/go/src/messageToDiscord
go build -o /usr/local/bin/message-to-discord .
```

## 2. systemd サービスファイルを作成する

```ini
[Unit]
Description=Discord mail relay for cron errors
After=network.target

[Service]
Type=simple
User=root
Group=root
Environment=DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
Environment=DISCORD_WEBHOOK_RETRIES=3
Environment=DISCORD_WEBHOOK_RETRY_DELAY=2
Environment=DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json
ExecStart=/usr/local/bin/message-to-discord
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

保存先例:

```bash
/usr/lib/systemd/system/message-to-discord.service
```

## 3. daemon-reload と起動

```bash
sudo systemctl daemon-reload
sudo systemctl enable message-to-discord.service
sudo systemctl start message-to-discord.service
sudo systemctl status message-to-discord.service
```

---

## 4. cron からの利用

`cron` 側では、通常のエラー出力をこのサービスに渡す構成にします。

```cron
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

* * * * * /usr/local/bin/your-job.sh 2>&1 | /usr/local/bin/message-to-discord
```

このとき、プログラムは `stdin` にジョブの標準出力とエラー出力を受け取り、ヘッダーを自動的に作成して Discord に送信します。

---

## 5. 注意事項

- `Environment=` に Webhook URL を含めるので、`crontab` 側で余計な設定は不要
- `Restart=on-failure` により、プロセスが落ちても再起動を試みる
- `DISCORD_FAILED_LOG_PATH` を設定しておくと、送信失敗時の情報を後から確認できる
- 実際の Webhook URL は秘密情報扱いにし、設定ファイルのアクセス権を制限する

---

## 6. 実運用時の推奨

- Webhook URL は環境変数または秘密管理下に置く
- `systemd` のログを確認して、失敗時のメッセージを監視する
- `journalctl -u message-to-discord` を見る
- 失敗ログファイルが一定量を超えたらローテーションする

```bash
journalctl -u message-to-discord -f
```
