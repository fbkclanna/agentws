# Development Guide

agentws の開発に参加するための技術ガイド。

## 前提条件

- Go 1.26.0 or later
- Git 2.25 or later
- golangci-lint
- lefthook
- GoReleaser

## Pre-commit hook

lefthook でコミット前に build, lint, test を自動実行する。

```bash
# インストール
go install github.com/evilmartians/lefthook@latest

# 有効化（.git/hooks/pre-commit にフックを設置）
lefthook install
```

## ビルド

```bash
# 開発ビルド
go build ./cmd/agentws

# バージョン埋め込みビルド
go build -ldflags "-X main.version=1.0.0" ./cmd/agentws

# クロスコンパイル（GoReleaser）
goreleaser release --snapshot --clean
```

## テスト

```bash
# 全テスト実行
go test ./...

# レースディテクタ付き
go test -race ./...

# カバレッジ付き
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# 特定パッケージ
go test ./internal/manifest/...
go test ./cmd/agentws/...
```

### テストの書き方

テストでは Git リポジトリが必要になることが多い。`internal/testutil` パッケージにヘルパーがある。

```go
import "github.com/fbkclanna/agentws/internal/testutil"

func TestSomething(t *testing.T) {
// 初期コミット付きの bare repo を作成（main ブランチ）
bare := testutil.CreateBareRepo(t)

// 追加ブランチ付きの bare repo を作成
bare := testutil.CreateBareRepoWithBranch(t, "feature/x")
}
```

CLI コマンドのテストでは `newRootCmd()` を使って cobra コマンドツリー全体を構築し、`SetArgs()` でフラグを渡す。

```go
func TestSomeCommand(t *testing.T) {
dir := t.TempDir()
root := newRootCmd()
root.SetArgs([]string{"--root", dir, "sync", "--jobs", "1"})
if err := root.Execute(); err != nil {
t.Fatal(err)
}
}
```

コマンドの標準出力をテストするには `cmd.SetOut()` で `bytes.Buffer` を設定する。出力先は必ず `cmd.OutOrStdout()` を使うこと（
`os.Stdout` は使わない）。

## リント

```bash
golangci-lint run
```

設定は `.golangci.yml` にある。有効なリンター: errcheck, govet, staticcheck, unused, ineffassign, gosimple, gofmt,
misspell。

## プロジェクト構造

```
agentws/
├── cmd/agentws/          # CLI エントリポイントと全サブコマンド
│   ├── main.go           # エントリポイント（version 変数）
│   ├── root.go           # ルートコマンド定義・サブコマンド登録
│   ├── cmd_init.go       # init サブコマンド
│   ├── cmd_sync.go       # sync サブコマンド
│   ├── cmd_status.go     # status サブコマンド
│   ├── cmd_pin.go        # pin サブコマンド
│   ├── cmd_branches.go   # branches サブコマンド
│   ├── cmd_checkout.go   # checkout サブコマンド
│   ├── cmd_start.go      # start サブコマンド
│   ├── cmd_doctor.go     # doctor サブコマンド
│   ├── cmd_run.go        # run サブコマンド
│   ├── templates.go      # init --template のテンプレート定義
│   └── exec.go           # post_sync コマンド実行ヘルパー
│
├── internal/
│   ├── manifest/         # workspace.yaml のモデルとパーサ
│   │   ├── model.go      # Workspace, Repo, Profile, Defaults 構造体
│   │   └── parse.go      # YAML パース・バリデーション・フィルタリング
│   │
│   ├── lock/             # workspace.lock.yaml のモデルとパーサ
│   │   ├── model.go      # File, Repo 構造体
│   │   └── parse.go      # Load/Parse/Save
│   │
│   ├── git/              # Git コマンドラッパー
│   │   └── git.go        # Clone, Fetch, Checkout, Branch 操作等
│   │
│   ├── workspace/        # ワークスペース操作コア
│   │   └── workspace.go  # Context, Load, Strategy
│   │
│   ├── ui/               # CLI 出力ユーティリティ
│   │   ├── table.go      # テーブル形式出力
│   │   └── progress.go   # 並列処理プログレス表示
│   │
│   └── testutil/         # テスト用ヘルパー
│       └── repo.go       # bare repo 作成ユーティリティ
│
├── .github/workflows/ci.yml   # GitHub Actions CI
├── .golangci.yml               # リンター設定
├── .goreleaser.yml             # リリース設定
└── workspace.yaml              # （ユーザーが作成する manifest の例）
```

## アーキテクチャ

### パッケージ依存関係

```
cmd/agentws
  ├── internal/manifest    （workspace.yaml の読み込み・フィルタリング）
  ├── internal/lock        （workspace.lock.yaml の読み書き）
  ├── internal/git         （Git 操作の実行）
  ├── internal/workspace   （manifest + lock の統合、パス解決）
  └── internal/ui          （テーブル・プログレス出力）

internal/workspace
  ├── internal/manifest
  └── internal/lock
```

`internal/git` は他の internal パッケージに依存しない。`internal/manifest` と `internal/lock` も互いに独立している。

### 設計方針

1. **defaults マージ**: repo 個別の設定が優先され、未指定なら `defaults` セクションの値を使う。`Repo.Effective*()`
   メソッドで解決する。

2. **Strategy パターン**: dirty な working tree の扱い方を `safe`（スキップ）/ `stash`（stash して続行）/ `reset`（強制リセット）の
   3 種から選択。`reset` は `--force` 必須。

3. **フィルタリング**: profile（タグ・ID ベース）と `--only`/`--skip`（ID ベース）の 2 層で repo を絞り込む。

4. **並列実行**: sync コマンドは goroutine + channel セマフォ（`--jobs` で制御）で並列に repo を処理する。`ui.Progress` が
   atomic カウンタ + mutex で安全に進捗を表示する。

5. **テスト容易性**: CLI 出力は `cmd.OutOrStdout()` 経由で書き出す。テストでは `cmd.SetOut(&buf)` でバッファにキャプチャできる。

## 新しいサブコマンドの追加方法

1. `cmd/agentws/cmd_<name>.go` を作成

```go
package main

import "github.com/spf13/cobra"

func newMyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mycmd",
		Short: "Description of my command",
		RunE:  runMyCmd,
	}
	// フラグ定義
	cmd.Flags().Bool("some-flag", false, "Flag description")
	return cmd
}

func runMyCmd(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	// workspace をロード
	ctx, err := workspace.Load(root)
	if err != nil {
		return err
	}
	// ロジック実装
	return nil
}
```

2. `root.go` の `newRootCmd()` に登録

```go
cmd.AddCommand(
// ...既存コマンド...
newMyCmd(),
)
```

3. テストを `cmd/agentws/cmd_<name>_test.go` に作成

## 共通フラグパターン

多くのサブコマンドで共通のフラグパターンを使用する。

| フラグ          | 型        | 用途                                |
|--------------|----------|-----------------------------------|
| `--root`     | string   | ワークスペースルートディレクトリ（persistent flag） |
| `--profile`  | string   | manifest の profile で repo をフィルタ   |
| `--only`     | []string | 指定した repo ID のみ対象                 |
| `--skip`     | []string | 指定した repo ID を除外                  |
| `--strategy` | string   | dirty tree の扱い（safe/stash/reset）  |
| `--force`    | bool     | 破壊的操作を許可                          |
| `--json`     | bool     | JSON 形式で出力                        |
| `--dry-run`  | bool     | 実行せずに計画を表示                        |

## バリデーションルール

manifest (`workspace.yaml`) には以下のバリデーションが適用される:

- `version` は `1` であること
- `name` は必須
- 各 repo の `id`, `url`, `path` は必須
- repo ID は一意であること
- `path` と `repos_root` は相対パスであること（絶対パス不可）
- `path` に `..` を含むパストラバーサルは不可

## CI/CD

GitHub Actions で以下を自動実行:

- **test**: `go build` → `go test -race -coverprofile` → カバレッジサマリー → artifact アップロード
- **lint**: `golangci-lint` による静的解析

リリースは GoReleaser で Linux/macOS/Windows × amd64/arm64 のバイナリを生成する。

## よくある注意点

- テスト内で Git リポジトリを操作する場合、bare repo の HEAD が意図しないブランチを指すことがある。
  `CreateBareRepoWithBranch` は clone 前に `main` に戻る処理が入っている。
- `os.Stdout` への直接書き込みはテストでキャプチャできない。必ず `cmd.OutOrStdout()` を使う。
- `git init` のような Git コマンドを実行する際、`cmd.Dir` に指定するディレクトリが存在していることを確認する。
- `--strategy reset` は `--force` なしでは実行を拒否する安全策がある。

## 推奨する GitHub リポジトリ設定

本番運用に向けた GitHub リポジトリ設定の推奨事項:

- **`main` ブランチの保護ルール**:
  - マージ前に Pull Request レビューを必須にする（最低 1 名の承認）
  - マージ前にステータスチェックのパスを必須にする（`test`, `lint`, `security`）
  - マージ前にブランチが最新であることを必須にする
  - force push を禁止する
- **Dependabot を有効にする**（`.github/dependabot.yml` で設定済み）
- **GitHub Actions を有効にする**（`.github/workflows/` で設定済み）
