# Petstore API テスト手順書

`runnora/practice/runbooks/generated/pet` の suite runbook を入力として作成したサンプル文書である。

このリポジトリは設計書の原稿である。原稿を書き、HTML で確認するところまでを
ここだけで行える。**PDF（発行版）は発行者が作る**ので、執筆者の環境には
Quarto 以外の道具は要らない（Node.js も不要）。

## 執筆者の準備（初回だけ・2つ）

1. **Quarto** を入れる … <https://quarto.org/docs/get-started/>
2. **VSCode の Quarto 拡張** を入れる … 拡張機能の検索窓で `Quarto`

このフォルダを VSCode で開けば、日本語を等幅で表示する設定
（`.vscode/settings.json`）も自動で効く。

## 書く

原稿は `docs/` の中だけである。

| 場所 | 中身 |
|---|---|
| `docs/index.qmd` | 前付け（改正履歴・用語など。採番されない） |
| `docs/chapters/` | 本文。章＝フォルダ／節＝サブフォルダ／項＝ファイル |
| `docs/diagrams/` | 静的な図の置き場 |

記法は**利用マニュアル**（発行者から配布される PDF / HTML）を参照する。

次のファイルは**様式の機構**であり、執筆者は触らない
（発行者がテンプレートの更新時に入れ替える）。

```
design-doc.lua  design-doc.css  postprocess-html.js  mermaid-config.json  .template-version
```

## 確認する（HTML）

VSCode で `.qmd` を開き、右上の **Preview**（`Ctrl+Shift+K`）を押す。
保存するたびに更新される。章番号・図表番号・相互参照は発行版の PDF と同じ
番号で表示される。

コマンドで出すこともできる（`docs/` の中で実行）。

```bash
quarto preview
```

design-doc-quarto-template の CLI を使う場合は、リポジトリのルートで次を実行する。

```powershell
ddq html docs
```

生成済み断片を更新する場合は、`runnora_test_iInstructions` のルートで次の形式を使用する。

```powershell
.\runnora-instructions.exe generate <Petstoreのsuite.yml...> --base-dir <runnoraのルート> --out examples\petstore-document\docs\generated --force
```

`docs/_book/index.html` に静的な HTML 一式が出る（`_book/` は
生成物なのでコミットしない）。`index.html` を直接開いて本文を読めるが、全文検索は
HTTP サーバから開いたときだけ使える。

## 発行版との違い

HTML で確認できないものが2つある。**発行者が定期的に作る「中間版」
（`docs/design-doc.pdf`。リポジトリに入っている）で確認する**こと。

- 紙の様式（外枠・資料番号欄・社名・ページ番号）と改ページ
- 横向きページ（`.landscape`）、表の分割（`.tbl` が複数ページに割れる形）

図（mermaid）は執筆者のブラウザで描かれ、発行版では同じ設定でベクター化される。
描画エンジンの版が違うため、まれに見え方が変わる。これも中間版で確認する。
