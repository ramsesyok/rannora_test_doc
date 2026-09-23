# 外部JSONを使う追加シナリオ

`runbooks/stock-recovery.yml`（ORD-004）と `runbooks/invalid-recovery.yml`（ORD-005）が使うデータです。
既存のORD-001〜003はYAMLへの直接記述の例として保持しています。

| フォルダ | 用途 |
|---|---|
| `requests/` | HTTPリクエスト本文。`empty.json` は初期化・取消用、`order-0.json` は数量不正、`order-1/2/3.json` は正常な注文 |
| `responses/` | 比較用の期待レスポンス本文。実測値の保存先ではありません |

パスは読み込むrunbookの場所を基準にします。runnoraが使うrunnの `json://` でJSONを読み込み、テンプレート式でリクエスト本文へ渡します。

```yaml
vars:
  req_order_1: json://../fixtures/requests/order-1.json
  res_confirmed_1: json://../fixtures/responses/confirmed-1.json
steps:
  create:
    req:
      /orders:
        post:
          body:
            application/json: "{{ vars.req_order_1 }}"
    test: |
      current.res.status == 201 &&
      current.res.body.id != "" &&
      compare(current.res.body, vars.res_confirmed_1, [".id"])
```

`compare` はJSONの構造と値を比較します。キーの並びや空白の違いでは失敗しません。
注文IDは毎回変わるため、注文の期待JSONは `sku`・`quantity`・`status` を保持し、`.id` だけを比較対象から外します。
IDは別の検証式で空でないことを確認して `bind` で保存し、照会・取消時のID一致と、複数注文間のIDの違いも確認します。
在庫・初期化・エラーの本文は、除外項目なしで全体を比較します。HTTPステータスは各runbookの検証式に明示しています。

ORD-004は注文2個＋3個で売切にし、追加注文の拒否、2個分の取消、1個の再注文、残る注文の取消を確認します。
ORD-005は数量0の入力エラー後に数量1でやり直し、照会・取消まで進みます。どちらも先頭で初期化し、単独実行・反復実行できます。

手順書にはJSONファイル名と実際のJSON内容を掲載し、元の検証式も残します。
期待値の詳細表にある「参照JSON」はファイルの内容です。比較で除外する項目や追加条件は検証式で確認してください。
JSONの内容を変更したら、シナリオを再実行し、`scripts/build-docs.ps1` で手順書も再生成してください。
