# 注文・在庫連携API：複数ステップのテストと手順書

既存のPetstore例に追加する、状態変化を追うサンプルです。
**まず [完成した手順書PDF](docs/design-doc.pdf) または [章別HTML](docs/_book/index.html) を開いてください。**
ddq 2.4.0で発行するQuarto book形式です。HTMLは再生成可能なためGit管理対象外です。

| シナリオ | 手順数 | 確認する流れ |
|---|---:|---|
| [ORD-001 正常系](runbooks/order-lifecycle.yml) | 9 | 在庫確認 → 注文 → ID保存 → 照会 → 在庫減算 → 取消 → 再照会 → 在庫復元（先頭に初期化） |
| [ORD-002 在庫不足](runbooks/insufficient-stock.yml) | 4 | 初期化 → 在庫確認 → 数量超過で409 → 在庫が変わらない |
| [ORD-003 二重キャンセル](runbooks/double-cancel.yml) | 8 | 初期化 → 在庫確認 → 注文 → ID保存 → 取消 → 再取消409 → 再照会 → 在庫が二重加算されない |
| [ORD-004 売切後の再注文](runbooks/stock-recovery.yml) | 20 | 2件で売切 → 追加注文409 → 1件取消 → 再注文 → 注文間の状態確認 → 全件取消 |
| [ORD-005 入力訂正](runbooks/invalid-recovery.yml) | 9 | 数量0で400 → 在庫不変 → 数量1で訂正注文 → 照会 → 取消 → 在庫復元 |

ORD-004・005は、リクエストと期待レスポンスを [fixtures/](fixtures/README.md) の外部JSONに分けた追加例です。
`vars` の `json://` で読み込み、本文を `compare` で検証します。採番IDは別に保存・検証します。
既存3件のrunbookは保持しています。全5件・50ステップ（HTTP 44回、ID保存6回）です。

## 見比べる順番

1. [OpenAPI定義](openapi.yaml)：4業務APIと教材用初期化APIの契約。
2. [OpenAPIからの生成雛形](baseline/runbooks/generated/)：API単位のtemplate／suite。
3. [実行用シナリオ](runbooks/)：業務順序・ID引き継ぎ・状態整合性を人が追加。
4. [生成されたQMD](docs/generated/)：同じYAMLを手順・HTTP・本文・検証式の表に変換。
5. [完成PDF](docs/design-doc.pdf)／[章別HTML](docs/_book/index.html)：目的、実施条件、実行・終了手順、結果欄と組み合わせた手順書。

## bookの構成

`docs/index.qmd` は採番しない前付けです。`docs/chapters/` の各章はフォルダを持ち、
`index.qmd` が節ごとのQMDをincludeします。章の順序は `docs/_quarto.yml` の `book.chapters` で管理します。

| 章フォルダ | 内容 |
|---|---|
| `01-overview/` | 試験対象・業務ルール・シナリオ一覧 |
| `02-source-workflow/` | OpenAPIとシナリオの対応、IDの引き継ぎ |
| `03-execution/` | 準備・実行・証跡・ddqでの発行・終了 |
| `04-order-lifecycle/` | 正常系の生成手順と詳細 |
| `05-insufficient-stock/` | 在庫不足の生成手順と詳細 |
| `06-double-cancel/` | 二重キャンセルの生成手順と詳細 |
| `07-stock-recovery/` | 外部JSONを使う売切後の再注文 |
| `08-invalid-recovery/` | 外部JSONを使う入力訂正 |
| `09-records/` | 実施記録 |

自動生成表は `docs/generated/` に保持し、章から相対パスでincludeします。
サンプル固有の `docs/workflow-layout.lua` が、第4〜8章を連続する横向きページにまとめます。
見出しだけの縦ページや、短い詳細表ごとの不要な改ページを防ぎ、生成原稿とddqの機構はそのまま使用します。
資料番号・会社名は未指定のため空欄です。発行時に `_quarto.yml` で設定できます。

OpenAPIだけから業務フローを自動推論したものではありません。業務シナリオを明示してから、
そのシナリオを手順書へ変換した例です。生成表の良否欄に実行結果は自動転記されません。

## 再実行

必要なもの：Go、runnora。文書発行にはQuartoとddq 2.4.0も必要です。DB・Java・Dockerは不要です。
リポジトリルートから `Set-Location examples/order-workflow` で移動します。

ターミナルA：

```powershell
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin/order-api.exe ./mock
if ($LASTEXITCODE -ne 0) { throw 'サンプルAPIのビルド失敗' }
./bin/order-api.exe
```

ターミナルB（同じフォルダ）：

```powershell
./scripts/run.ps1
./scripts/build-docs.ps1
```

スクリプトは隣接する `runnora/runnora.exe` を既定で使用します。
別配置なら `-Runnora C:/tools/runnora.exe` を指定してください。
文書発行用ddqは隣接する `design-doc-quarto-template/cli/target/release/ddq.exe` が既定です。
配布環境では `-Ddq C:/tools/quarto-template-2.4.0/ddq.exe` を指定してください。
`build-docs.ps1` は既定でHTMLとPDFの両方を作り、`-Format Html`／`-Format Pdf` で出力を選択できます。
機構更新が必要な場合だけ `-UpdateTemplate` を指定します。通常の発行では版・内容の不一致をddqが検出して停止します。
接続先は `127.0.0.1:18081`。同じAPIプロセスに対する並列実行はしません。
各シナリオは先頭で在庫と注文を初期化するので、単独・反復実行できます。
終了時はターミナルAで `Ctrl+C`。結果は `reports/<日時>/` に残ります。

生成雛形のbaselineは比較用で、完成した実行入口は `scripts/run.ps1` と `runbooks/` です。
このサンプルは実APIの品質証明ではありません。
確認した内容と制約は [作成時の検証記録](verification.md)、詳しい実施手順は手順書第3章を参照してください。

## 原稿だけを編集・発行する

執筆者は `docs/` で `quarto preview` を実行すればHTMLを確認できます。
発行者はサンプルのルートで次を実行します（実行ファイルのパスは環境に合わせて変更）。

```powershell
$ddq = '../../../design-doc-quarto-template/cli/target/release/ddq.exe'
& $ddq pdf docs
if ($LASTEXITCODE -ne 0) { throw 'PDF発行失敗' }
& $ddq html docs
if ($LASTEXITCODE -ne 0) { throw 'HTML発行失敗' }
```

PDFは `docs/design-doc.pdf`、HTMLは `docs/_book/` です。HTML配布時はフォルダ一式を渡してください。
ddq 2.4.0は形式ごとに `_book/` を作り直します。両方残す場合はPDF→HTMLの順で発行してください。
PDFの横向き手順表、表の分割、外枠・ページ番号はPDFで確認します。
