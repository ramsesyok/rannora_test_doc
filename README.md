# runnora-instructions

## 新人向けチュートリアル

API単体試験のファイル構成、正常系・異常系の作成、実行、テスト手順書への変換を一通り学べる教材です。

- [導入プレゼンテーション（PowerPoint・14枚）](docs/tutorial/output/runnora-api-testing.pptx)
- [Step by Step 詳細手順（HTML）](docs/tutorial/output/step-by-step.html)
- [教材の使い方・完成例](docs/tutorial/README.md)

## このツールについて

runnora 用の runn シナリオを、レビュー用の Quarto `.qmd` 原稿へ変換する Go CLI です。
生成する表は `design-doc-quarto-template` の `::: {.landscape}` と `::: {.tbl}`、
Pandoc GridTable を使用し、PDFでは常に横向きページとして組版します。

## 対応範囲

- HTTP
- gRPC Unary RPC / Server streaming RPC
- suite、include、case JSON
- runnora 設定およびコマンドで指定する前後処理 SQL/PLSQL
- 検証式と期待ステータスの文書化

Client streaming RPC と双方向 streaming RPC は初期版の対象外です。テスト実行や実測レスポンスの収集も行いません。

## コマンド

```powershell
go run . generate `
  --out C:\work\test-document\generated `
  --config C:\work\scenario\config.yaml `
  --before-sql C:\work\scenario\sql\setup.sql `
  --proto C:\work\scenario\proto\service.proto `
  C:\work\scenario\runbooks\scenario.yml
```

主なオプション:

| オプション | 内容 |
|---|---|
| `-o, --out` | 生成原稿の出力ディレクトリ（既定: `generated`） |
| `-c, --config` | runnora 設定 YAML |
| `--before-sql` | 追加の前処理 SQL。複数指定可能 |
| `--after-sql` | 追加の後処理 SQL。複数指定可能 |
| `--proto` | RPC 種別判定に使う `.proto`。複数指定可能 |
| `--base-dir` | config と追加 SQL の相対パス、および生成物に記録する入力元パスの基準（既定: 現在のディレクトリ） |
| `-f, --force` | 既存の生成原稿を上書き |

シナリオごとに `scenario.qmd`、`cases.qmd`、`http.qmd`、`grpc.qmd`、
`request-json.qmd`、`grpc-request.qmd`、`expectations.qmd`、前後処理原稿を必要に応じて生成します。
既存文書の章立てや `_quarto.yml` は変更しません。

設計判断と今後の検討事項は [docs/design-notes.md](docs/design-notes.md) を参照してください。
