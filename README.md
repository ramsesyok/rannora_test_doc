# runnora-docgen

GitHubリポジトリ: [ramsesyok/runnora-docgen](https://github.com/ramsesyok/runnora-docgen)

CLI名は `runnora-docgen` です。試験実行ツール `runnora` のシナリオを文書化します。

## 新人向けチュートリアル

API単体試験のファイル構成、正常系・異常系の作成、実行、テスト手順書への変換を一通り学べる教材です。

- [導入プレゼンテーション（PowerPoint・14枚）](docs/tutorial/output/runnora-api-testing.pptx)
- [Step by Step 詳細手順（HTML）](docs/tutorial/output/step-by-step.html)
- [教材の使い方・完成例](docs/tutorial/README.md)

## 複数ステップのサンプル

- [注文・在庫連携APIのサンプル](examples/order-workflow/README.md)：注文IDの引き継ぎ、在庫の増減、在庫不足、二重キャンセル、外部JSONを使う売切後の再注文・入力訂正を確認する5シナリオ。
- [テスト手順書（PDF）](examples/order-workflow/docs/design-doc.pdf) ／ [章別HTML](examples/order-workflow/docs/_book/index.html)：ddqで発行するQuarto book形式。実行用YAMLから生成した表と実施手順を確認できます。

## このツールについて

runnora 用の runn シナリオを、レビュー用の Quarto `.qmd` 原稿へ変換する Go CLI です。
生成する表は `design-doc-quarto-template` の `::: {.landscape}` と `::: {.tbl}`、
Pandoc GridTable を使用し、PDFでは常に横向きページとして組版します。

## 対応範囲

- HTTP
- gRPC Unary RPC / Server streaming RPC
- suite、include、case JSON（include の `vars` に書いた `json://` も読み込む。後述）
- runnora 設定およびコマンドで指定する前後処理 SQL/PLSQL
- 検証式と期待ステータスの文書化

Client streaming RPC と双方向 streaming RPC は初期版の対象外です。テスト実行や実測レスポンスの収集も行いません。

## コマンド

リポジトリのルートでビルドし、実行します。

```powershell
go build -o runnora-docgen.exe .
.\runnora-docgen.exe generate `
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
前後処理原稿（`before.qmd`・`after.qmd`）は実行順・ファイル名・出典パスの表で、SQL 本文は掲載しません（表が大きくなり読みにくいため。本文は出典のファイルを参照）。
既存文書の章立てや `_quarto.yml` は変更しません。

include ステップの `vars` に `json://` で渡したファイルは、runn と同じく include 先 runbook の位置を基準に読み込みます。
include 先の URL・ヘッダ・リクエストボディにある `{{ vars.xxx }}` を実際の値に置き換え、
検証式の期待ステータス（例：`vars.case.expect.status`）を求め、参照しているファイル名を手順表に追記します。
期待値表には、検証式が変数ごと参照するファイル（例：`compare(..., vars.expected)`）は全体を、
一部の項目だけ参照するファイル（例：`vars.case.expect.status`）はその項目だけを載せます。
1 ステップだけの template を include する場合は、include ステップの `desc` を手順名にします。

シナリオ表では、HTTP呼び出し情報を `URL：表 4.2-1-[1]`、期待値を
`期待値：[200] 表 4.4-1` の形式で参照します（番号は配置先の章・節に従います）。
リクエストボディ・期待値がケースJSONや `vars` の `json://...json` を参照する場合は、
詳細表への参照の次行にJSONファイル名だけを追記します。複数ある場合は改行して列挙します。
YAMLへの直接記述や、実行時にしか決まらないファイル名には追記しません。
ステータス未指定時は値を補わず、ケースごとに異なる場合は `[ケース別]`、gRPCは `[gRPC 0]` 等で示します。

設計判断と今後の検討事項は [docs/design-notes.md](docs/design-notes.md) を参照してください。
