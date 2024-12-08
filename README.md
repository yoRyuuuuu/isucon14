### TODO

- [] SSHでサーバーに接続する
- [] サーバーのIPアドレスを設定する
  - `/etc/hosts`
- [] チーム全員の公開鍵を取得するように変更する
  - `ansible/roles/install_tools/ssh-key.yml`
- [] 計測結果を通知するGitHubのリポジトリを設定する
  - `scripts/notify_result.sh`
- [] `make install_tools`を実行し、サーバーにツールをインストールする
- [] サーバー上でGitHunの認証を行う
  - `gh auth login`
- [] GitHubにソースコードをプッシュする
  - アプリケーションサーバーのソースコード
  - テーブル定義のSQLファイル
  - その他の設定ファイル

### hostsファイルの設定

`/etc/hosts`に以下の設定を追加する。

```sh
XXX.XXX.XXX.XXX isucon1
XXX.XXX.XXX.XXX isucon2
XXX.XXX.XXX.XXX isucon3
XXX.XXX.XXX.XXX bench
```

### ツールのインストール

```sh
make install_tools
```

### ベンチマーク準備

```sh
make before_bench
```
