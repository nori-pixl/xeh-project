package core

import (
	"fmt"
	"sync"
)

// Context は各ハンドラーに渡される実行コンテキスト。
// main.go が組み立て、internal/handlers 配下の各ハンドラーがこれを使って
// 共有メモリ・プロセス管理・set.json の設定にアクセスする。
type Context struct {
	// TagName は現在処理中のXMLタグ名 (例: "xeh-logic", "python")
	TagName string

	// MemSpaceName は app.xeh の <root memory_name="..."> で指定された名前
	MemSpaceName string

	// Store は全エンジン共有のメモリ空間
	Store *SharedStore

	// Processes は起動した外部プロセスの一覧
	Processes *ProcessList

	// WaitGroup は非同期で起動した外部エンジンの終了待ち合わせ用
	WaitGroup *sync.WaitGroup

	// SetConfig は set.json の内容そのもの (設計図)
	SetConfig *SetConfig
}

// HandlerFunc はタグごとの処理を表す関数の型。
// node には対象タグのXML情報、ctx には実行コンテキストが渡される。
type HandlerFunc func(node XehNode, ctx *Context) error

var (
	registryMu sync.RWMutex
	registry   = make(map[string]HandlerFunc)
)

// Register はハンドラーIDに対応する処理関数を登録する。
// 各ハンドラーファイルの init() から呼び出す想定。
//
//	例: core.Register("core.logic", RunLogic)
func Register(id string, fn HandlerFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("xeh: handler '%s' is already registered", id))
	}
	registry[id] = fn
}

// Get は登録済みハンドラーをIDで取得する
func Get(id string) (HandlerFunc, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := registry[id]
	return fn, ok
}

// List は登録済みハンドラーID一覧を返す (デバッグ・診断用)
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}
