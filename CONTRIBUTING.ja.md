# agentws へのコントリビュート

> See [CONTRIBUTING.md](CONTRIBUTING.md) for the English version.

agentws へのコントリビュートに興味を持っていただきありがとうございます！バグ報告、機能リクエスト、ドキュメント改善、コード変更など、あらゆる種類のコントリビュートを歓迎します。

## コントリビュートの方法

### バグ報告

- 重複を避けるため、まず[既存の issue](https://github.com/fbkclanna/agentws/issues) を検索してください。
- [Bug Report テンプレート](https://github.com/fbkclanna/agentws/issues/new?template=bug_report.md)を使い、再現手順・期待動作・環境情報を記載してください。

### 機能提案

- [Feature Request テンプレート](https://github.com/fbkclanna/agentws/issues/new?template=feature_request.md)を使って issue を作成してください。
- 解決したい課題と提案する解決策を記載してください。

### Pull Request の提出

1. リポジトリを fork し、`main` から新しいブランチを作成します。
2. [DEVELOPMENT.ja.md](DEVELOPMENT.ja.md) の手順に従って開発環境をセットアップします。
3. 変更を行い、テストがパスすることを確認します:
   ```sh
   go build ./cmd/agentws
   go test -race ./...
   golangci-lint run
   ```
4. 必要に応じてテストを追加・更新します。
5. 変更内容を明確に説明した Pull Request を作成します。

## 開発環境セットアップ

ビルド・テスト・リントの詳細は [DEVELOPMENT.ja.md](DEVELOPMENT.ja.md) を参照してください。

## コーディング規約

- Go の標準規約に従う（`gofmt`、`go vet`）。
- 関数は焦点を絞り、十分にテストする。
- CLI 出力には `cmd.OutOrStdout()` を使用する（`os.Stdout` を直接使わない）。
- ユーザー入力のパスには必ずバリデーションを行う（絶対パスや `..` トラバーサルの禁止）。

## コミットメッセージ規約

[Conventional Commits](https://www.conventionalcommits.org/) を採用しています:

| プレフィックス | 用途                |
|---------|-------------------|
| `feat:`  | 新機能               |
| `fix:`   | バグ修正              |
| `docs:`  | ドキュメントのみの変更       |
| `test:`  | テストの追加・更新         |
| `chore:` | メンテナンス、CI、依存関係    |
| `ci:`    | CI/CD 設定の変更       |

例: `feat: checkout コマンドに --dry-run フラグを追加`

## ライセンス

コントリビュートいただいた内容は [MIT License](LICENSE) の下でライセンスされます。
