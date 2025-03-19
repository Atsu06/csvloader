package main

import (
	"csvloader/csvloader"
	"fmt"
	"log"
	"os"
)

func main() {
	// サンプルCSVデータ
	csvFile := "example.csv"

	// ファイルを開く
	file, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	// CSVをパース
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
