package handlers

import (
	"fmt"
	"strings"

	"xeh/internal/core"
)

func init() {
	core.Register("core.logic", RunLogic)
}

// RunLogic は <xeh-logic> タグ内のネイティブxeh命令(現在は <print>)を実行する。
//
// set.json側では以下のように指定する:
//
//	"handlers": { "xeh-logic": "core.logic" }
func RunLogic(node core.XehNode, ctx *core.Context) error {
	fmt.Println("[xeh/os] Executing native xeh XML logic:")

	for _, child := range node.Children {
		switch child.XMLName.Local {
		case "print":
			runPrint(child, ctx)
		default:
			fmt.Printf("   [xeh Warning]: unknown instruction <%s>\n", child.XMLName.Local)
		}
	}

	return nil
}

// runPrint は <print value="..."> または <print>...</print> を出力する。
// ${変数名} は共有メモリの値に置換される。
func runPrint(node core.XehNode, ctx *core.Context) {
	output := node.Value
	if output == "" {
		output = node.Content
	}

	ctx.Store.Range(func(k string, v interface{}) {
		output = strings.ReplaceAll(output, "${"+k+"}", fmt.Sprintf("%v", v))
	})

	fmt.Printf("   [xeh Print]: %s\n", strings.TrimSpace(output))
}
