# Discord へ cron エラー通知を転送するプログラム

## 概要

このプログラムは、cron やその他の送信元が通常のメール送信のように呼び出す `sendmail` 互換の入口を受け取り、メール本文を解析して Discord の Webhook に送信する Go 製の小さな転送サービスです。

主な目的は、cron の失敗通知や監視ジョブのエラーを、メールとして受け取り、Discord 上で確認できるようにすることです。

---

## 仕組み

### 1. 送信元からの呼び出し

送信元は通常のメール送信のように、プログラムにメール本文を標準入力として渡します。

典型的な入力例:

```bash
echo "job failed\nline 42: permission denied" | go run . -s "Cron <root@host> test job" -f "root@host"
```

ここで `-s` は件名、`-f` は送信元です。

### 2. メール解析

プログラムは標準入力から本文を受け取り、以下の情報を抽出します。

- `From`
- `To`
- `Subject`
- `Date`
- `Body`

また、`MIME` と `Content-Transfer-Encoding` を考慮して、複雑なメール本文も処理できるようにしています。

### 3. Discord Webhook 送信

解析したメールを、Discord へ送るための JSON に変換します。

- 件名をタイトルにする
- 送信元を本文に含める
- 本文を抜粋して表示する
- 長い本文は省略する

### 4. エラー時の運用対応

運用時に必要になる次の振る舞いを持っています。

- Webhook の一時失敗に対する再試行
- 失敗時に payload を保存する機能
- 送信の失敗内容をログに出す

これにより、Webhook の一時障害やネットワークブレイクにも比較的安定して対応できます。

---

## 使用方法

### 1. 環境変数の設定

実行時に必要な設定は、それぞれの実行環境で設定します。

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
export DISCORD_WEBHOOK_RETRIES=3
export DISCORD_WEBHOOK_RETRY_DELAY=2
export DISCORD_FAILED_LOG_PATH="/tmp/discord_failed.json"
```

### 2. 手動実行

```bash
cd /home/chouette/go/src/messageToDiscord
go run . -s "Cron <root@host> backup failed" -f "root@host" <<EOF
backup job failed
Permission denied
EOF
```

### 3. cron からの利用

`crontab` に以下のように記述します。

```cron
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
DISCORD_WEBHOOK_RETRIES=3
DISCORD_WEBHOOK_RETRY_DELAY=2
DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json

* * * * * /usr/local/bin/your-job.sh 2>&1 | /usr/local/bin/sendmail-to-discord
```

`/usr/local/bin/sendmail-to-discord` は、この Go プログラムを呼び出すラッパーとして使います。

### 4. systemd での利用

```ini
[Service]
Environment=DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
Environment=DISCORD_WEBHOOK_RETRIES=3
Environment=DISCORD_WEBHOOK_RETRY_DELAY=2
Environment=DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json
ExecStart=/usr/local/bin/sendmail-to-discord
```

---

## 前提条件

### 1. 送信元はメールのような入力を渡すこと

このプログラムは `sendmail` の代替として使う前提です。つまり、送信元が以下のような入力を与える前提で設計されています。

- 標準入力に本文を渡す
- `-s` で件名を渡す
- `-f` で送信元を渡す
- 必要に応じて `To` を付与できる

### 2. Webhook URL を持っていること

Discord の Webhook URL は必須です。

- 取得済みであること
- Webhook の権限が有効であること
- URL が漏れないよう管理されていること

### 3. 実行環境に GOLANG が入っていること

Go で実行する前提なので、実行環境に Go が入っている必要があります。

---

## 制約

### 1. sendmail 互換の最小実装である

これは完全な MTA ではありません。

- 実際のメール配送サーバーのような複雑なルーティングは行わない
- 受信側の SMTP の仕様を完全には実装しない
- 送信元のルールやメールキューは扱わない

### 2. メールの複雑な構造には限定的

主要なケースは以下です。

- 通常のテキストメール
- 日本語を含むメール
- 簡単な multipart メール
- base64 / quoted-printable の本文

ただし、非常に複雑な MIME 構造や大量の添付ファイルにはこの実装が制約を受けます。

### 3. Discord の本文には長さ制限がある

Discord は本文サイズに上限があります。そこで、本文は短く要約して送るようにしています。

- 長い本文は省略される
- 内容全体をそのまま表示できない
- 添付ファイルの本文をそのまま転送しない

### 4. Webhook 送信失敗時の対処は自前で行う

このプログラムは送信に失敗した場合、再試行とログ保存を行いますが、完全な配送保証を行うものではありません。

- 送信タイムアウト
- 一時的な Webhook 障害
- Discord 側の制限

といった事象に対しては、運用側での対策が必要です。

---

## 例外と改善余地

今後の改善候補としては次があります。

- `attachment` を Discord のファイル送信として扱う
- `multipart/report` や複雑な MIME の更なる対応
- `logrus` や `slog` によるロギング強化
- `retry` の指数バックオフ調整
- `mailq` のような配信キューの導入

---

## 実務上の判断

今回の目的である「cron の失敗通知を Discord で見たい」というケースには、この実装で十分に達成できます。

特に、以下のような使い方が想定されています。

- cron が失敗した
- エラー内容がメールとして流れる
- このプログラムがそれを拾う
- Discord に通知される

これは、小規模な監視運用において非常に有効です。

---

## まとめ

このプログラムは、`sendmail` のようなメール入力を受け取り、本文を解析して Discord に転送する軽量な橋渡しです。

本来の電子メール配送の仕組みを完全に再現するものではありませんが、cron の失敗通知の運用用途には十分適しています。

必要に応じて、次は実際の cron 設定例とシステム構成図を追加することもできます。
