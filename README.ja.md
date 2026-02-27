# agentws

[![CI](https://github.com/fbkclanna/agentws/actions/workflows/ci.yml/badge.svg)](https://github.com/fbkclanna/agentws/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/v/release/fbkclanna/agentws)](https://github.com/fbkclanna/agentws/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/fbkclanna/agentws)](https://goreportcard.com/report/github.com/fbkclanna/agentws)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/fbkclanna/agentws/pulls)

> [English](README.md)

複数リポジトリに分割されたプロダクト（backend / frontend / infra / analytics …）を、再現性高くローカルに揃えるための CLI
ツールです。

Codex や Claude Code などのコーディングエージェントと組み合わせることで、複数 repo を同一 workspace
配下に統一配置し、正しいディレクトリから起動できます。

## 特徴

- プロダクトごとに "正しい repo セット" を **ワンコマンドで clone/sync**
- チームで同じ workspace 構成を共有でき、手作業の差分や漏れを防止
- `workspace.lock.yaml` により同一 commit 群での再現が可能（バグ再現・検証・レビューに最適）
- `profile` で必要な repo だけを選択的に取得（重い分析基盤 repo などを除外可能）

## インストール

### クイックインストール (Linux / macOS)

```sh
curl -fsSL https://github.com/fbkclanna/agentws/releases/latest/download/agentws-install.sh | sh
```

バージョンやインストール先を指定する場合:

```sh
VERSION=0.2.0 INSTALL_DIR=~/.local/bin \
  curl -fsSL https://github.com/fbkclanna/agentws/releases/latest/download/agentws-install.sh | sh
```

### Go install

```sh
go install github.com/fbkclanna/agentws/cmd/agentws@latest
```

### GitHub Releases

[Releases ページ](https://github.com/fbkclanna/agentws/releases) からプラットフォーム別のバイナリをダウンロードできます（Linux / macOS / Windows、amd64 / arm64）。

**要件:**

- Git 2.25 以上（sparse-checkout サポートのため）
- Go 1.26 以上（`go install` を使う場合）

### アンインストール

`agentws-install.sh` または GitHub Releases でインストールした場合、バイナリを削除します:

```sh
sudo rm /usr/local/bin/agentws
```

`INSTALL_DIR` を指定してインストールした場合は、そのディレクトリから削除します:

```sh
rm ~/.local/bin/agentws
```

`go install` でインストールした場合:

```sh
rm $(go env GOPATH)/bin/agentws
```

## クイックスタート

```sh
# 1) workspace を作る（対話的にリポジトリを追加）
agentws init <workspace-name>

# 2) repo を揃える（同期）
agentws sync

# 3) 状態を確認
agentws status
```

## ディレクトリ構成

デフォルトでは `--root` 配下に workspace を作ります（例: `./products`）。

```
products/
└── foo/
    ├── workspace.yaml
    ├── workspace.lock.yaml
    ├── AGENTS.md
    ├── CLAUDE.md -> AGENTS.md
    ├── docs/
    │   └── agentws-guide.md
    └── repos/
        ├── backend/
        ├── frontend/
        ├── infra/
        └── analytics/
```

## コマンド

### `init <name>`

新しい workspace を作り、`workspace.yaml`、`AGENTS.md`、`CLAUDE.md`（`AGENTS.md` へのシンボリックリンク）を生成します。

オプションなしで実行すると対話モードが起動し、リポジトリの URL とブランチを順次入力できます。
URL からリポジトリ ID とパスを自動推定し、リモートのデフォルトブランチも検出します。URL を空のまま Enter を押すとローカルリポジトリ（リモートなし）を追加できます。

```sh
agentws init foo
# リモートリポジトリ:
# ? Enter Git repository URL (empty for local): git@github.com:org/backend.git
#   → id: backend, path: repos/backend
# ? Branch: main
# ? Add another repository? Yes
#
# ローカルリポジトリ（URL を空で Enter）:
# ? Enter Git repository URL (empty for local): [Enter]
# ? Enter repository name (ID): config
#   → id: config, path: repos/config (local)
# ? Add another repository? No
```

既存の manifest ファイルから作成する場合は `--from` を使います:

```sh
agentws init foo --from git@github.com:org/workspaces.git#foo.yaml
```

**オプション:**

| オプション          | 説明                                           |
|----------------|----------------------------------------------|
| `--root <dir>` | workspace を作るルート（例: `./products`）            |
| `--from <src>` | manifest を取り込む（例: ローカルパス、または `repo#path` 形式） |
| `--force`      | 既存 workspace があっても上書き（注意）                    |

### `add [url ...]`

既存の workspace にリポジトリを追加します。CLI モードと対話モードの両方に対応しています。

```sh
# CLI モード: URL を指定して追加
agentws add https://github.com/org/backend.git
agentws add https://github.com/org/api.git --id api-service --ref develop --tag core

# ローカルリポジトリモード: リモートなしのリポジトリを作成
agentws add --local my-service
agentws add --local my-service --path custom/dir --ref main --tag core
agentws add --local my-service --sync   # git init + 初期コミットまで実行

# 対話モード: URL なしで実行
agentws add
```

URL を指定せず stdin が TTY の場合、対話モードが起動します（`init` と同じインターフェース）。URL を空のまま Enter を押すとローカルリポジトリも追加できます。

**オプション:**

| オプション | 説明 |
|--------|-------------|
| `--local` | ローカルリポジトリを作成（リモート URL なし）。引数は ID として扱う |
| `--id <string>` | リポジトリ ID を上書き（URL 1個のみ有効） |
| `--path <string>` | リポジトリパスを上書き（1リポジトリのみ有効） |
| `--ref <string>` | チェックアウトする ref（デフォルト: 自動検出、ローカルは `main`） |
| `--tag <string>` | リポジトリに付与するタグ（複数指定可） |
| `--sync` | 追加後に即座にクローン/初期化 |
| `--json` | 追加されたリポジトリを JSON で出力 |

### `sync`

`workspace.yaml` に従って、repo を clone/fetch/checkout して workspace を揃えます。
冪等です（何度実行しても状態が揃うことを目指しています）。

```sh
agentws sync
```

**よく使うオプション:**

| オプション              | 説明                   |
|--------------------|----------------------|
| `--profile <name>` | profile に従い repo を選択 |
| `--jobs <n>`       | 並列処理数（例: `8`）        |
| `--only <id1,id2>` | 指定 repo のみ同期         |
| `--skip <id1,id2>` | 指定 repo を除外          |

**再現性（lock）:**

| オプション           | 説明                                                    |
|-----------------|-------------------------------------------------------|
| `--lock`        | `workspace.lock.yaml` の commit に固定して checkout（再現性モード） |
| `--update-lock` | 同期後に lock を更新                                         |

**作業ツリーが dirty のときの扱い:**

| `--strategy`  | 説明                          |
|---------------|-----------------------------|
| `safe`（デフォルト） | dirty repo はスキップ            |
| `stash`       | stash して続行                  |
| `reset`       | 強制 reset（通常 `--force` 必須推奨） |

**安全/破壊的操作:**

| オプション     | 説明                          |
|-----------|-----------------------------|
| `--force` | 破壊的操作を許可（`reset` などとセットで使用） |

### `status`

workspace の状態を一覧表示します。

- 未 clone / clone 済み
- 現在の HEAD
- dirty 判定
- lock/manifest との差分（ある場合）

```sh
agentws status
```

**オプション:**

| オプション    | 説明               |
|----------|------------------|
| `--json` | JSON 出力（CI 連携向け） |

### `pin`

現在の各 repo の HEAD を `workspace.lock.yaml` に固定（commit を記録）します。

```sh
agentws pin foo
```

### `branches`

workspace 配下の各リポジトリについて、現在のブランチ・HEAD コミット・作業ツリー状態（dirty）を一覧表示します。
マルチ repo 横断開発で「いま各 repo がどのブランチ/どのコミットか」を素早く確認できます。

```sh
agentws branches
```

**出力例:**

```
REPO        BRANCH                      HEAD        DIRTY
backend     feature/ABC-123-search-v2   a1b2c3d     false
frontend    feature/ABC-123-search-v2   c3d4e5f     true
infra       main                        9f8e7d6     false
analytics   (detached)                  deadbeef    false
```

**オプション:**

| オプション              | 説明                      |
|--------------------|-------------------------|
| `--json`           | JSON 形式で出力              |
| `--profile <name>` | profile に含まれる repo のみ対象 |
| `--only <id1,id2>` | 指定 repo のみ対象            |
| `--skip <id1,id2>` | 指定 repo を除外             |

> **補足:**
> - `BRANCH` が `(detached)` の場合、特定コミットを checkout した状態です（lock 同期や commit 指定 checkout など）。
> - `DIRTY=true` は未コミット変更（tracked/untracked を含む）がある状態です。

### `checkout --branch <branch>`

workspace 配下の対象 repo を、同じブランチ名に一括で切り替えます。

- ローカルにブランチが存在する場合 → そのまま checkout
- ローカルに無いが remote にある場合 → tracking branch を作成して checkout
- どちらにも無い場合 → フラグに従い新規作成またはスキップ

```sh
agentws checkout --branch feature/ABC-123-search-v2
```

**オプション:**

| オプション                           | 説明                            |
|---------------------------------|-------------------------------|
| `--create`                      | ブランチが存在しない場合に新規作成             |
| `--from <ref>`                  | 新規作成時の起点（`base_ref` を上書き）      |
| `--profile <name>`              | profile に含まれる repo のみ対象       |
| `--only <id1,id2>`              | 指定 repo のみ対象                  |
| `--skip <id1,id2>`              | 指定 repo を除外                   |
| `--strategy safe\|stash\|reset` | dirty 時の扱い（デフォルト `safe`）      |
| `--force`                       | 破壊的操作を許可（`reset` とセットで使用）     |
| `--dry-run`                     | 実行せず、対象 repo と操作内容を表示         |

### `start <ticket> [slug]`

チケット ID から命名規約に沿ったブランチ名を生成し、workspace 配下の対象 repo を一括でブランチ作成＆checkout します。
横断機能開発で「全 repo を同名ブランチに揃える」作業をワンコマンドで実行できます。

```sh
agentws start ABC-123 search-v2
# => feature/ABC-123-search-v2 を作成＆checkout
```

**オプション:**

| オプション                              | 説明                          |
|------------------------------------|-----------------------------|
| `--prefix feature\|bugfix\|hotfix` | ブランチ種別（デフォルト `feature`）     |
| `--from <ref>`                     | 作成起点（`base_ref` を上書き）        |
| `--profile <name>`                 | profile に含まれる repo のみ対象     |
| `--only <id1,id2>`                 | 指定 repo のみ対象                |
| `--skip <id1,id2>`                 | 指定 repo を除外                 |
| `--strategy safe\|stash\|reset`    | dirty 時の扱い                  |
| `--force`                          | 破壊的操作を許可                    |
| `--dry-run`                        | 実行せず、生成されるブランチ名と対象 repo を表示 |

> **補足:**
> - remote に同名ブランチが存在する repo はそれを checkout します（tracking branch を作成）。
> - ブランチが存在しない repo は `--from` または `origin/<base_ref>` を起点に新規作成します。
> - 起点の解決順序: `--from` フラグ → `origin/<repo.base_ref>` → `origin/<defaults.base_ref>` → エラー。

### `doctor`

開発環境の診断を行います。問題がある場合はエラーを報告します。

```sh
agentws doctor
```

**チェック項目:**

- Git のインストール確認
- Git バージョンチェック
- SSH 認証チェック（`ssh -T git@github.com`）
- workspace 内の各 repo URL への接続テスト（`git ls-remote`）

### `run -- <command>`

workspace ルートディレクトリでコマンドを実行します。`--` 以降がそのまま実行されます。

```sh
agentws run -- make test
agentws run -- docker compose up -d
```

## Manifest: `workspace.yaml`

`workspace.yaml` は「このプロダクト workspace を構成する repo とルール」を宣言します。

### 例

```yaml
version: 1
name: foo
description: Foo product workspace
repos_root: repos

profiles:
  core:
    include_tags: [ "core" ]
  full:
    include_tags: [ "core", "infra", "data" ]

defaults:
  depth: 50
  partial_clone: false
  sparse_checkout: false
  base_ref: main

repos:
  - id: backend
    url: git@github.com:org/foo-backend.git
    path: repos/backend
    ref: main
    tags: [ "core" ]
    depth: 50
    post_sync:
      - name: "go mod download"
        workdir: "."
        cmd: [ "go", "mod", "download" ]

  - id: analytics
    url: git@github.com:org/foo-analytics.git
    path: repos/analytics
    ref: main
    tags: [ "data" ]
    base_ref: develop
    partial_clone: true
    sparse:
      - "pipelines/"
      - "docs/"

  - id: config
    local: true
    path: repos/config
    ref: main
    tags: [ "core" ]
```

### フィールド

#### workspace

| フィールド         | 説明                         |
|---------------|----------------------------|
| `version`（必須） | 現在は `1`                    |
| `name`（必須）    | workspace 名                |
| `description` | 説明                         |
| `repos_root`  | repo を置くルート（デフォルト `repos`） |

#### defaults

| フィールド             | 説明                                        |
|-------------------|-------------------------------------------|
| `depth`           | shallow clone 深さ（例: `50`）                                           |
| `partial_clone`   | blob を取らない clone（`--filter=blob:none` 相当）                           |
| `sparse_checkout` | sparse checkout の既定                                                 |
| `base_ref`        | `start`/`checkout --create` のデフォルト起点ブランチ（ブランチ名のみ。例: `main`） |

#### profiles

| フィールド              | 説明                |
|--------------------|-------------------|
| `include_tags`     | 指定タグを持つ repo を含める |
| `include_repo_ids` | 明示的に含める repo id   |
| `exclude_repo_ids` | 明示的に除外する repo id  |

#### repo

| フィールド                              | 説明                                           |
|------------------------------------|----------------------------------------------|
| `id`（必須）                           | 論理名（ユニーク）                                    |
| `url`                              | git URL（リモート repo は必須、ローカル repo は空であること）      |
| `local`                            | `true` でローカルリポジトリ（リモート URL なし）                |
| `path`（必須）                         | clone 先（相対パス。絶対パス / `..` 禁止）                 |
| `ref`                              | branch/tag/commit（省略時は `main`）               |
| `base_ref`                         | `start`/`checkout --create` の起点（`defaults.base_ref` を上書き） |
| `tags`                             | profile 用タグ                                  |
| `required`                         | `true`/`false`（省略時 `true`）                   |
| `depth`, `partial_clone`, `sparse` | 個別設定                                         |
| `post_sync`                        | 同期後に実行するコマンド（配列）。`cmd` は配列で指定（shell 展開なしで安全） |

## Lock: `workspace.lock.yaml`

`workspace.lock.yaml` は「実際に揃えた commit」を記録し、再現性を担保します。

### 例

```yaml
version: 1
name: foo
generated_at: "2026-02-15T12:34:56+09:00"
tool_version: "0.1.0"

repos:
  backend:
    url: git@github.com:org/foo-backend.git
    ref: main
    commit: "a1b2c3d4..."
  analytics:
    url: git@github.com:org/foo-analytics.git
    ref: main
    commit: "deadbeef..."
```

**使い方:**

```sh
# lock を作る/更新する
agentws sync --update-lock

# lock の commit に固定して揃える
agentws sync --lock
```

## 安全性（重要）

この CLI は workspace 外への書き込みを防ぐため、`workspace.yaml` の `repos_root` / `repos[].path`
に対して以下を禁止します:

- 絶対パス
- `..` を含むパス

また、`--strategy reset` のような破壊的操作は `--force` と併用する運用を推奨します。

## 典型的なワークフロー（チーム運用例）

### 1) workspace 定義 repo から配布する

workspace 定義をまとめた repo（例: `org/workspaces`）に `foo.yaml` を置きます。
メンバーは以下で揃えます:

```sh
agentws init foo --from git@github.com:org/workspaces.git#foo.yaml
agentws sync --profile core
```

### 2) バグ再現や検証は lock で固定する

```sh
agentws sync --lock
```

### 3) 検証用に lock を更新する

```sh
agentws sync --update-lock
```

## コントリビュート

コントリビュート大歓迎です！ガイドラインは [CONTRIBUTING.ja.md](CONTRIBUTING.ja.md) をご覧ください。

> See [CONTRIBUTING.md](CONTRIBUTING.md) for the English version.

## 開発

詳細は [DEVELOPMENT.ja.md](DEVELOPMENT.ja.md) を参照してください。

```sh
# ビルド
go build ./cmd/agentws

# テスト
go test -race ./...

# リント
golangci-lint run
```

## License

MIT License. 詳細は [LICENSE](LICENSE) を参照してください。
