package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"xeh/internal/core"
)

// LoadRootVariables は app.xeh の <root> 内のJSON配列をパースし、
// 共有メモリ空間(SharedStore)に初期値として書き込む。
// メインの app.xeh 読み込み時に1回だけ呼ばれる。
//
//	<root memory_name="default_space">
//	  [{"name":"count","type":"int","value":0}]
//	</root>
func LoadRootVariables(jsonStr string, store *core.SharedStore, memSpaceName string) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return
	}

	var variables []core.XehVariable
	if err := json.Unmarshal([]byte(jsonStr), &variables); err != nil {
		log.Fatalf("[Error] Failed to parse root JSON: %v", err)
	}

	fmt.Printf("[xeh/os] Allocating memory space: '%s'\n", memSpaceName)
	for _, v := range variables {
		store.Set(v.Name, v.Value)
		fmt.Printf("   -> %s = %v\n", v.Name, v.Value)
	}
}

// MergeRootVariables はインポートされたプラグイン(.xeh)の <root> JSONを
// 既存の共有メモリ空間にマージする。エラーは警告のみで処理を継続する。
func MergeRootVariables(jsonStr string, store *core.SharedStore) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return
	}

	var variables []core.XehVariable
	if err := json.Unmarshal([]byte(jsonStr), &variables); err != nil {
		log.Printf("[Warning] Failed to parse imported root JSON: %v", err)
		return
	}

	for _, v := range variables {
		store.Set(v.Name, v.Value)
	}
}
