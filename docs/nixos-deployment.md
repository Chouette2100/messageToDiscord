# NixOS に配置して使う方法

## 1. 基本方針

Ubuntu では `sendmail` のようなコマンドを `/usr/local/bin` や `/usr/sbin` に配置して使うのが一般的ですが、NixOS ではその方法は通常推奨されません。

NixOS では、システムの構成は `nix` の宣言的設定で管理するのが基本です。つまり、このプログラムは、

- Nix 管理下の実行ファイルとして配置する
- `systemd` のサービスとして起動する
- 必要に応じて `environment.systemPackages` や `systemd.services` で登録する

という形が普通です。

---

## 2. NixOS での配置パターン

### パターン A: システムパッケージとして用意する

`/etc/nixos/configuration.nix` に、Go でビルドしたバイナリを追加する場合の考え方です。

```nix
{ config, pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    # このプログラムを Nix 管理下で使う場合は、nix package として定義する
  ];
}
```

ただし、この補助プログラムは通常、専用の Nix package として定義した方が管理しやすいです。

---

## 3. 専用 package として定義する例

以下は、プログラムを Nix package として定義するイメージです。

```nix
{ pkgs, ... }:
let
  messageToDiscord = pkgs.stdenvNoCC.mkDerivation {
    pname = "message-to-discord";
    version = "0.1.0";

    src = /home/chouette/go/src/messageToDiscord;

    buildInputs = [ pkgs.go ];

    installPhase = ''
      mkdir -p $out/bin
      go build -o $out/bin/message-to-discord .
    '';
  };
in {
  environment.systemPackages = [ messageToDiscord ];
}
```

この場合、実行ファイルは `message-to-discord` として利用可能になります。

---

## 4. sendmail 互換のコマンド名で提供する

NixOS でも `sendmail` として使いたいなら、シンボリックリンクまたは wrapper を用意する方法があります。

```nix
{ pkgs, ... }:
let
  app = pkgs.writeShellScriptBin "sendmail" ''
    exec ${pkgs.messageToDiscord}/bin/message-to-discord "$@"
  '';
in {
  environment.systemPackages = [ app ];
}
```

上の例では、`sendmail` という名前で実行できるようにしています。

---

## 5. systemd サービスとして置く

NixOS では `systemd.services` を用いて、長時間運用のアプリとして動かすのが普通です。

```nix
{ config, pkgs, ... }:
{
  systemd.services.message-to-discord = {
    description = "Discord relay for cron/mail alerts";
    wantedBy = [ "multi-user.target" ];
    after = [ "network.target" ];
    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.messageToDiscord}/bin/message-to-discord";
      Environment = [
        "DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/..."
        "DISCORD_WEBHOOK_RETRIES=3"
        "DISCORD_WEBHOOK_RETRY_DELAY=2"
        "DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json"
      ];
      Restart = "on-failure";
      RestartSec = 5;
    };
  };
}
```

なお、ここでは「常時待ち受ける」サービスではなく、実際には `stdin` を受け取る 一回限りのプログラムとして使うものです。

そのため、NixOS では一般に「sendmail 相当のコマンド」を提供するファイルとして設置し、必要なときに呼ぶ形が自然です。

---

## 6. cron からの利用

NixOS でも cron は同様に使えます。

```cron
SHELL=/run/current-system/sw/bin/bash
PATH=/run/current-system/sw/bin:/run/current-system/sw/sbin:/nix/var/nix/profiles/default/bin:/nix/var/nix/profiles/default/sbin
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
DISCORD_WEBHOOK_RETRIES=3
DISCORD_WEBHOOK_RETRY_DELAY=2

* * * * * /run/current-system/sw/bin/your-job.sh 2>&1 | /run/current-system/sw/bin/sendmail
```

NixOS の `PATH` は通常 `/run/current-system/sw` 配下になっているので、そこに入っているバイナリが安定して使いやすいです。

---

## 7. 環境変数の設定方法

### 方法 A: `systemd.services` の `Environment`

```nix
Environment = [
  "DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/..."
  "DISCORD_WEBHOOK_RETRIES=3"
  "DISCORD_WEBHOOK_RETRY_DELAY=2"
  "DISCORD_FAILED_LOG_PATH=/tmp/discord_failed.json"
];
```

### 方法 B: `environment.sessionVariables`

ログインシェル向けに使う場合:

```nix
environment.sessionVariables = {
  DISCORD_WEBHOOK_URL = "https://discord.com/api/webhooks/...";
  DISCORD_WEBHOOK_RETRIES = "3";
  DISCORD_WEBHOOK_RETRY_DELAY = "2";
  DISCORD_FAILED_LOG_PATH = "/tmp/discord_failed.json";
};
```

### 方法 C: cron の中で明示

```cron
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
DISCORD_WEBHOOK_RETRIES=3
DISCORD_WEBHOOK_RETRY_DELAY=2
```

---

## 8. NixOS での注意点

### 注意点 1: `/usr/local` に手書きで置かない

NixOS では、手動で `/usr/local/bin` に置く運用は推奨されません。宣言的管理に寄せるのが基本です。

### 注意点 2: /etc/nixos/configuration.nix を編集する

システム全体の構成は Nix に載せるのが自然です。

### 注意点 3: 再構成時に反映される

NixOS では設定を反映するために:

```bash
sudo nixos-rebuild switch
```

を使います。

---

## 9. まとめ

NixOS では、Ubuntu のように `/usr/local/bin` に直接配置するのではなく、

- Nix package として定義する
- systemd で管理する
- `sendmail` 互換のラッパーを用意する

という形が自然です。

本プログラムの用途そのものは変わらず、cron やジョブの失敗通知を Discord に転送する機能はそのまま使えます。

ただし、NixOS では「システムのファイルを手で置く」より「Nix で宣言する」方が運用に合います。

---

## 10. 実用例の整理

以下のどれかの形が NixOS での標準運用に近いです。

1. `message-to-discord` を package として定義し、`sendmail` 互換の wrapper を用意する
2. `systemd` のサービス定義と `cron` の連携を組み合わせる
3. `nix-shell` や `nix run` で一時的に使う

最も実務的なのは、1 と 2 を組み合わせる方法です。
