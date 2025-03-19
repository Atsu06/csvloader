# csvloader

ほぼ自分用です

## 特徴

- CSVファイルを簡単にJSONに変換
- ヘッダー名と行数からデータの取り出し
- Golangの標準ライブラリのみを使用

## インストール

以下のコマンドで `csvloader` をインストールできます。

```sh
go get github.com/Atsu06/csvloader
```

## 使い方

### 基本的な使用例

```go
package main

import (
    "fmt"
    "log"
    "github.com/Atsu06/csvloader"
)

func main() {
	// サンプルCSVデータ
	csvFile := "example.csv"

	// CSVを開く
	data, err := csvloader.OpenCSV(csvFile, "utf-8")
	if err != nil {
		log.Fatalf("failed to load CSV: %v", err)
	}

	// JSONへ変換
	jsonData, err := data.ToJSON()
	if err != nil {
		log.Fatalf("failed to conver to JSON: %v", err)
	}
	// 結果を出力
	fmt.Println(jsonData)

	// 行数と列名からstringポインタ要素を取り出し
	var s string
	s, err = data.GetString(0, "name")
	if err != nil {
		log.Fatalf("failed to GetStringPtr: %v", err)
	}
	// 結果を出力
	fmt.Println(s)

	// 行数と列名からstringポインタ要素を取り出し
	var sPtr *string
	sPtr, err = data.GetStringPtr(0, "name")
	if err != nil {
		log.Fatalf("failed to GetStringPtr: %v", err)
	}
	// 結果を出力
	fmt.Println(*sPtr)
}
```

### CSVフォーマット例

```csv
name,age,city
Alice,30,Tokyo
Bob,25,Osaka

```

### JSON出力例

```json
[
  {
    "age": "30",
    "city": "Tokyo",
    "name": "Alice"
  },
  {
    "age": "25",
    "city": "Osaka",
    "name": "Bob"
  }
]
```
