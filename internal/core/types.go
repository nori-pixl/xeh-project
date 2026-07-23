package core

import "encoding/xml"

// ===== app.xeh (XML) のパース用構造体 =====

// XehNode は <subframework> 内の任意のタグを表す汎用ノード
type XehNode struct {
	XMLName  xml.Name
	Value    string    `xml:"value,attr"`
	Content  string    `xml:",chardata"`
	Children []XehNode `xml:",any"`
}

// SubFramework は <subframework> タグ
type SubFramework struct {
	RoteApp   string    `xml:"roteapp,attr"`
	ImportKey string    `xml:"import,attr"`
	Nodes     []XehNode `xml:",any"`
}

// XehRoot は <root> タグ。共有メモリの初期値(JSON)を保持する
type XehRoot struct {
	MemoryName string `xml:"memory_name,attr"`
	Content    string `xml:",chardata"`
}

// XehServer は <server> タグ
type XehServer struct {
	ID       string `xml:"id,attr"`
	Port     string `xml:"port,attr"`
	InnerXML string `xml:",innerxml"`
}

// XehApp は app.xeh 全体(<xeh> ルート要素)
type XehApp struct {
	XMLName      xml.Name       `xml:"xeh"`
	Root         XehRoot        `xml:"root"`
	Servers      []XehServer    `xml:"server"`
	SubFramework []SubFramework `xml:"subframework"`
}

// XehVariable は <root> 内のJSON配列1要素分の変数定義
type XehVariable struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// MemoryPacket は外部エンジン(子プロセス)にstdin経由で渡す共有メモリのスナップショット
type MemoryPacket struct {
	MemoryName string                 `json:"memory_name"`
	Data       map[string]interface{} `json:"data"`
}

// ===== set.json (設計図) のパース用構造体 =====

// MetaSetting はアプリのメタ情報
type MetaSetting struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	Charset string `json:"charset"`
	Mail    string `json:"mail"`
}

// RuntimeSetting は外部言語を実行するコマンド定義 (例: python, node)
type RuntimeSetting struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// EngineSetting はタグごとの外部エンジン設定 (どの言語で・どのファイルに書き出すか)
type EngineSetting struct {
	Type string `json:"type"` // Runtimesのキーに対応 (例: "python")
	Src  string `json:"src"`  // 書き出し先ファイルパス
}

// PluginSetting はインポート可能な外部 .xeh ファイルの設定
type PluginSetting struct {
	Src string `json:"src"`
}

// SetConfig は set.json 全体。これが「設計図」本体。
//
//	Handlers: タグ名 -> ハンドラーID (内部レジストリに登録された処理の識別子)
//	Engines:  タグ名 -> 外部エンジン設定 (core.engine ハンドラーが参照する)
type SetConfig struct {
	Meta     MetaSetting               `json:"meta"`
	Runtimes map[string]RuntimeSetting `json:"runtimes"`
	Engines  map[string]EngineSetting  `json:"engines"`
	Plugins  map[string]PluginSetting  `json:"plugins"`
	Handlers map[string]string         `json:"handlers"`
}
