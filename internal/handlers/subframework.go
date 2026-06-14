package handlers

import (
	"log"

	"xeh/internal/core"
)

// DispatchSubFrameworks は app.xeh 内の全 <subframework> を処理する。
//
//  1. import属性があれば set.json の "plugins" からプラグインをロードし、
//     その <root> 変数を共有メモリにマージする。
//  2. 子ノードそれぞれを set.json の "handlers" 設計図に従ってディスパッチする。
func DispatchSubFrameworks(subs []core.SubFramework, ctx *core.Context) {
	for _, sub := range subs {
		if sub.ImportKey != "" {
			importPlugin(sub.ImportKey, ctx)
		}
		for _, node := range sub.Nodes {
			DispatchNode(node, ctx)
		}
	}
}

// importPlugin は set.json の "plugins" で定義された .xeh ファイルを読み込み、
// その <root> 変数を共有メモリ空間にマージする。
func importPlugin(importKey string, ctx *core.Context) {
	pluginConfig, exists := ctx.SetConfig.Plugins[importKey]
	if !exists {
		log.Printf("[Warning] Plugin key '%s' is not defined in set.json.", importKey)
		return
	}

	importedApp, err := core.LoadXehFile(pluginConfig.Src)
	if err != nil {
		log.Printf("[Error] %v", err)
		return
	}

	MergeRootVariables(importedApp.Root.Content, ctx.Store)
}

// DispatchNode は1タグ(node)について、set.jsonの "handlers" マップを参照し、
// 対応するハンドラーをレジストリから取得して実行する。
//
// "handlers" に未定義でも "engines" に同名タグの定義があれば、
// 汎用外部エンジンハンドラー "core.engine" にフォールバックする。
//
//	"handlers": {
//	  "xeh-logic": "core.logic",
//	  "python":    "core.engine"
//	}
func DispatchNode(node core.XehNode, ctx *core.Context) {
	tagName := node.XMLName.Local

	handlerID, ok := ctx.SetConfig.Handlers[tagName]
	if !ok {
		if _, exists := ctx.SetConfig.Engines[tagName]; exists {
			handlerID = "core.engine"
		} else {
			log.Printf("[Warning] No handler defined for tag '<%s>' in set.json (handlers)", tagName)
			return
		}
	}

	fn, ok := core.Get(handlerID)
	if !ok {
		log.Printf("[Error] Handler '%s' for tag '<%s>' is not registered in the binary", handlerID, tagName)
		return
	}

	// タグごとにコンテキストをコピーしてTagNameだけ差し替える
	nodeCtx := *ctx
	nodeCtx.TagName = tagName

	if err := fn(node, &nodeCtx); err != nil {
		log.Printf("[Error] Handler '%s' for tag '<%s>' failed: %v", handlerID, tagName, err)
	}
}
