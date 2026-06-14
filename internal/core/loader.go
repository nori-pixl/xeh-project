package core

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
)

// LoadSetConfig は set.json (設計図) を読み込みパースする。
// 設計図自体が読めない場合はプログラムを継続できないため致命的エラーとする。
func LoadSetConfig(path string) *SetConfig {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("[Error] Failed to open %s: %v", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("[Error] Failed to read %s: %v", path, err)
	}

	var cfg SetConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("[Error] Failed to parse %s: %v", path, err)
	}

	return &cfg
}

// MustLoadXehFile はメインの app.xeh を読み込む。失敗した場合はプログラムを終了する。
func MustLoadXehFile(path string) XehApp {
	app, err := LoadXehFile(path)
	if err != nil {
		log.Fatalf("[Error] %v", err)
	}
	return app
}

// LoadXehFile は .xeh ファイル(XML)を読み込みパースする。
// プラグインなど、失敗してもプログラムを継続したい場合はこちらを使い、
// 呼び出し側でエラーをハンドリングする。
func LoadXehFile(path string) (XehApp, error) {
	f, err := os.Open(path)
	if err != nil {
		return XehApp{}, fmt.Errorf("failed to open xeh file '%s': %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return XehApp{}, fmt.Errorf("failed to read xeh file '%s': %w", path, err)
	}

	var app XehApp
	if err := xml.Unmarshal(data, &app); err != nil {
		return XehApp{}, fmt.Errorf("failed to parse xeh file '%s': %w", path, err)
	}

	return app, nil
}
