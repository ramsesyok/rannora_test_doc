# runnora API単体試験 新人向け教材

導入資料で全体像を学び、詳細手順で自分のテスト資産と手順書を作成します。

| 資料 | 内容 |
|---|---|
| [導入プレゼンテーション](output/runnora-api-testing.pptx) | 14枚、説明の目安15〜20分。PowerPointで編集可能 |
| [詳細チュートリアル](output/step-by-step.html) | Step 0〜11。Windows／PowerShell、演習60〜90分の目安 |
| [講師用ノート](slide-notes.md) | 各スライドの説明と補足 |
| [詳細手順の原稿](step-by-step.qmd) | 改訂用のQuarto原稿 |
| [演習用API仕様](assets/openapi.yaml) | GET /pet/{petId}、200と404の応答 |
| [完成版suite](assets/runbooks/evidence/pet/get_getPetById.suite.yml) | 正常系・異常系を明示した実行入口 |
| [手順書の雛形](assets/document/docs/index.qmd) | 実施条件、生成原稿のinclude、結果記録欄 |

演習は詳細手順のStep 0から始めてください。完成例のassetsは比較用です。最初から一括コピーする必要はありません。`config.yaml` は演習中に生成します。文書表示用の機構はStep 9で既存サンプルからコピーします。

教材はモックでテスト作成・文書化の流れを練習します。実APIの品質を判定する試験結果ではありません。DBやgRPCは応用の説明のみで、基礎演習の必須環境ではありません。

## 改訂する場合

`step-by-step.qmd` を編集し、このフォルダで次を実行します。

```powershell
quarto render step-by-step.qmd --output step-by-step.html
if ($LASTEXITCODE -ne 0) { throw 'HTML生成に失敗しました' }
Copy-Item step-by-step.html output/step-by-step.html -Force
```

HTMLはCSS等を埋め込んだ単一ファイルです。説明の閲覧だけなら `output/step-by-step.html` 単体を配布できます。演習を行う人には、このリポジトリの `docs/tutorial/assets/` と既存の `examples/petstore-document/` も使える状態で配布してください。

導入スライドはPPTXを直接編集できます。教材を更新する場合は、例のcase・suite・template、講師用ノート、詳細手順の内容も合わせて確認します。

## この版で反映した動作

確認日：2026-09-22。参照ソースの版は詳細手順末尾に記載しています。

- 生成直後のtemplateはステータスのみを比較するため、本文の比較式を追加しています。
- 確認したrunnoraのレポート実装に合わせ、`--report-format text` と `.txt` を使用しています。
- 確認したバイナリでは生成suiteのloop形式で先行ケースの失敗が最終ケースの成功に隠れる挙動があったため、完成版はケース別のincludeステップを使っています。
- 基礎演習の成功、正常系本文の意図的な不一致による失敗、値を戻した後の成功、QMD生成、手順書HTML生成を確認しています。

配布時には使用するrunnoraの版をそろえてください。API仕様・検証式・実行条件を変更したら、同じ確認を行ってから教材を更新します。
