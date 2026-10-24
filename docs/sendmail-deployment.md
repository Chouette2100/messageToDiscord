# sendmail として配置して使う方法

## 1. 前提

このプログラムは、`sendmail` の代替として配置して使う想定です。

つまり、通常の MTA 連携のように、システムがメール送信を行うときに本プログラムの実行ファイルを呼び出し、標準入力にメール本文を渡す構成を前提にしています。

本プログラムは、以下のような「送信側の振る舞い」を受け取る想定です。

- 標準入力に本文を渡す
- `-s` で件名を受け取る
- `-f` で送信元を受け取る
- `To` や `Cc` を必要に応じて扱う
- 送信結果を終了コードで返す

実際には完全な MTA ではないため、一般的な完全な sendmail の機能は持っていませんが、以下の用途では十分に使えます。

- cron の失敗通知
- ジョブのエラー出力の通知
- 監視ジョブからのアラート送信

---

## 2. sendmail として配置する場所

一般に、システムが `sendmail` を探す場所は以下のようなパスです。

- `/usr/sbin/sendmail`
- `/usr/lib/sendmail`
- `/usr/local/bin/sendmail`
- `/usr/local/sbin/sendmail`

実運用では、以下のように配置するのが普通です。

```bash
sudo cp /home/chouette/go/src/messageToDiscord/message-to-discord /usr/local/bin/sendmail
sudo chmod 755 /usr/local/bin/sendmail
```

もし実際に `sendmail` コマンド名を使いたいなら、リンクを作る方法もあります。

```bash
sudo ln -sf /usr/local/bin/message-to-discord /usr/local/bin/sendmail
```

> 重要: システムが `PATH` の中で `sendmail` を見つける場所に置くことが前提です。

---

## 3. sendmail の代替としての振る舞い

本プログラムは最低限、以下のような入力を受け取ると動くようにしています。

```bash
printf 'job failed\npermission denied\n' | /usr/local/bin/sendmail -f root@localhost -s 'cron failed'
```

または、メール形式のまま標準入力に渡す方法もあります。

```bash
cat <<'EOF' | /usr/local/bin/sendmail
From: root@localhost
To: alert@example.com
Subject: cron failed

job failed
permission denied
EOF
```

このとき、プログラムは入力を解析して、Discord に転送します。

---

## 4. sendmail 互換の制約

完全な sendmail ではないため、制約があります。

### 制約 1: 完全な MTA ではない

- 送信元/宛先のルーティングを行わない
- メールキューを持たない
- 送信先の SMTP 連携はしない
- 受信側の配送を保証しない

### 制約 2: 実運用では一部の sendmail オプションに限定される

この実装では、最低限の互換性があります。

- `-f` で From を指定
- `-s` で Subject を指定
- `-t` や `-oi` を無視する形で扱う
- 標準入力に本文を読む

本格的な sendmail の機能を完全には再現していません。

### 制約 3: 本文の複雑さに限界がある

- 長いメール本文は省略される
- マルチパート形式は簡易対応
- 実際の巨大なメール体制には向かない

---

## 5. シェルスクリプトでのテスト方法

### 例 1: 簡単なテスト

```bash
#!/bin/bash
set -eu

echo "cron failed\npermission denied" | /usr/local/bin/sendmail \
  -f root@localhost \
  -s "Cron job failed"
```

このスクリプトを実行すると、本文が Discord に送信されます。

### 例 2: 本文をメール形式で渡す

```bash
#!/bin/bash
set -eu

cat <<'EOF' | /usr/local/bin/sendmail
From: root@localhost
To: alert@example.com
Subject: Cron backup failed

backup script failed
permission denied
EOF
```

### 例 3: cron 風のエラーを模擬する

```bash
#!/bin/bash
set -eu

(
  echo "backup failed"
  echo "line 42: permission denied"
  echo "stderr sample"
) | /usr/local/bin/sendmail \
  -f root@localhost \
  -s "Cron <root@localhost> backup job"
```

---

## 6. 実際の確認方法

### 1. Webhook URL が正しいか確認する

```bash
echo $DISCORD_WEBHOOK_URL
```

### 2. テストメールを投げる

```bash
printf 'hello from sendmail\n' | /usr/local/bin/sendmail -f root@localhost -s 'test message'
```

### 3. Discord で確認する

送信されたメッセージが本文の抜粋や件名として表示されるか確認します。

### 4. 失敗時のログを確認する

```bash
cat /tmp/discord_failed.json
```

---

## 7. 実運用の着地

システムが `sendmail` を呼ぶような環境では、このプログラムを `sendmail` として配置して使うことで、エラー通知を Discord に転送できます。

ただし、以下は前提条件です。

- `sendmail` として呼び出すバイナリが配置されている
- Webhook URL が設定されている
- 送信側が標準入力に本文を渡す
- 途中の制約を許容する

---

## 8. まとめ

このプログラムは完全な `sendmail` ではなく、`sendmail` らしい入力を受けて Discord に転送する小型の変換器です。

そのため、以下のような用途に向いています。

- cron エラー通知
- 監視ジョブの通知
- 警告メールの Discord 化

完全なメール配送基盤ではないため、運用の前提と制約を理解したうえで使うのが安全です。
