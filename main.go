package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"xeh/internal/core"
	"xeh/internal/handlers"
)

func main() {
	// set.json = 設計図。タグ名 -> ハンドラーID のマッピングなどを保持する。
	setConfig := core.LoadSetConfig("set.json")

	xehFileName, handled := handleCLIArgs(setConfig)
	if handled {
		return
	}

	fmt.Printf("--- %s (Version %s / %s License / %s) ---\n",
		setConfig.Meta.Name, setConfig.Meta.Version, setConfig.Meta.License, setConfig.Meta.Charset)

	mainApp := core.MustLoadXehFile(xehFileName)

	// --- 共有メモリ空間の初期化 (<root>) ---
	store := core.NewSharedStore()
	memSpaceName := mainApp.Root.MemoryName
	if memSpaceName == "" {
		memSpaceName = "default_space"
	}
	handlers.LoadRootVariables(mainApp.Root.Content, store, memSpaceName)

	// --- 子プロセス管理 & シグナルハンドラ ---
	processes := core.NewProcessList()
	setupSignalHandler(processes)

	// --- HTTPサーバー (<server>) ---
	for _, srv := range mainApp.Servers {
		handlers.StartHTTPServer(srv, store)
	}

	// --- <subframework> 以下を set.json の設計図に従ってディスパッチ ---
	var wg sync.WaitGroup
	ctx := &core.Context{
		MemSpaceName: memSpaceName,
		Store:        store,
		Processes:    processes,
		WaitGroup:    &wg,
		SetConfig:    setConfig,
	}
	handlers.DispatchSubFrameworks(mainApp.SubFramework, ctx)

	// 非同期で起動した外部エンジン(言語プロセス)の終了を待つ
	wg.Wait()

	// HTTPサーバーが起動している場合はプロセスを維持し続ける
	if len(mainApp.Servers) > 0 {
		select {}
	}
}

// handleCLIArgs は --version / --license / run / new などのCLI引数を処理する。
// 戻り値の1つ目は main が読み込むべき .xeh ファイル名 (run未指定時は "app.xeh")。
// 2つ目が true の場合、その場で処理が完結しているので main はそのまま終了する。
func handleCLIArgs(cfg *core.SetConfig) (string, bool) {
	if len(os.Args) <= 1 {
		return "app.xeh", false
	}

	switch strings.ToLower(os.Args[1]) {
	case "--version", "-v":
		fmt.Printf("%s version %s\n", cfg.Meta.Name, cfg.Meta.Version)
		return "", true
	case "--license":
		fmt.Printf("%s is licensed under the %s License.\n", cfg.Meta.Name, cfg.Meta.License)
		return "", true
	case "--help", "-h":
		fmt.Printf("--version, --v\n")
		fmt.Printf("--license\n")
		fmt.Printf("--help, -h\n")
		fmt.Printf("--mail\n")
		fmt.Printf("--config\n")
		fmt.Printf("run <xehファイル名>\n")
		fmt.Printf("new project <フォルダ名> <作成場所>\n")
		fmt.Printf("new plugin <プラグインのファイル名>\n")
		fmt.Printf("new xehfile <ファイル名>\n")
		return "", true
	case "--mail":
		fmt.Printf("gmail: sato.shigure4@gmail.com")
		return "", true
	case "--config":
		fmt.Printf("set.json")
		return "", true
	
	case "run":
		return handleRunCommand()
	case "new":
		return "", handleNewCommand()
		// switchの最後に追加する
	default:
		fmt.Printf("No command available.: %s\n", os.Args[1])
		fmt.Println("To check how to use it, run -help.")
		return "", true
	}

}

// handleRunCommand は `xeh run <名前>` を処理する。
// 指定された .xeh ファイル名を返し、main側でそのまま読み込ませる。
func handleRunCommand() (string, bool) {
	if len(os.Args) < 3 {
		fmt.Println("使い方: xeh run <動かすxehの名前>")
		return "", true
	}

	name := os.Args[2]
	if !strings.HasSuffix(strings.ToLower(name), ".xeh") {
		name += ".xeh"
	}

	if _, err := os.Stat(name); err != nil {
		fmt.Printf("[Error] xehファイルが見つかりません: %s\n", name)
		return "", true
	}

	return name, false
}

// handleNewCommand は `xeh new project|plugin|xehfile ...` を振り分ける。
func handleNewCommand() bool {
	if len(os.Args) < 3 {
		fmt.Println("使い方: xeh new project|plugin|xehfile ...")
		return true
	}

	switch strings.ToLower(os.Args[2]) {
	case "project":
		return newProjectCommand()
	case "plugin":
		return newPluginCommand()
	case "xehfile":
		return newXehFileCommand()
	default:
		fmt.Printf("不明な new サブコマンドです: %s\n", os.Args[2])
		return true
	}
}

// newProjectCommand は `xeh new project <フォルダ名> <作成場所>` を処理する。
// 作成場所の下にフォルダ名のディレクトリを作り、最低限の set.json / app.xeh / plugins を用意する。
func newProjectCommand() bool {
	if len(os.Args) < 5 {
		fmt.Println("使い方: xeh new project <作りたいフォルダ名> <作る場所>")
		return true
	}
	folderName := os.Args[3]
	location := os.Args[4]

	projectPath := filepath.Join(location, folderName)

	if _, err := os.Stat(projectPath); err == nil {
		fmt.Printf("[Error] すでに存在します: %s\n", projectPath)
		return true
	}

	if err := os.MkdirAll(filepath.Join(projectPath, "plugins"), 0755); err != nil {
		fmt.Printf("[Error] プロジェクトフォルダの作成に失敗しました: %v\n", err)
		return true
	}

	if err := os.WriteFile(filepath.Join(projectPath, "set.json"), []byte(defaultSetJSON(folderName)), 0644); err != nil {
		fmt.Printf("[Error] set.json の作成に失敗しました: %v\n", err)
		return true
	}

	if err := os.WriteFile(filepath.Join(projectPath, "app.xeh"), []byte(defaultXehTemplate()), 0644); err != nil {
		fmt.Printf("[Error] app.xeh の作成に失敗しました: %v\n", err)
		return true
	}

	fmt.Printf("新しいプロジェクト '%s' を %s に作成しました。\n", folderName, projectPath)
	return true
}

// newPluginCommand は `xeh new plugin <ファイル名>` を処理する。
// plugins/ フォルダ配下に、拡張子に応じた雛形ファイルを作成する。
func newPluginCommand() bool {
	if len(os.Args) < 4 {
		fmt.Println("使い方: xeh new plugin <追加するプラグインのプログラムのファイル名>")
		return true
	}
	fileName := os.Args[3]

	if err := os.MkdirAll("plugins", 0755); err != nil {
		fmt.Printf("[Error] plugins フォルダの作成に失敗しました: %v\n", err)
		return true
	}

	pluginPath := filepath.Join("plugins", fileName)
	if _, err := os.Stat(pluginPath); err == nil {
		fmt.Printf("[Error] すでに存在します: %s\n", pluginPath)
		return true
	}

	if err := os.WriteFile(pluginPath, []byte(defaultPluginTemplate(fileName)), 0644); err != nil {
		fmt.Printf("[Error] プラグインファイルの作成に失敗しました: %v\n", err)
		return true
	}

	fmt.Printf("新しいプラグイン %s を作成しました。\n", pluginPath)
	fmt.Println("set.json の \"engines\" / \"handlers\" (必要なら \"runtimes\") への登録も忘れずに行ってください。")
	return true
}

// newXehFileCommand は `xeh new xehfile <ファイル名>` を処理する。
// カレントディレクトリに最低限の .xeh 雛形ファイルを作成する。
func newXehFileCommand() bool {
	if len(os.Args) < 4 {
		fmt.Println("使い方: xeh new xehfile <新しく作りたいxehのファイル名>")
		return true
	}
	fileName := os.Args[3]
	if !strings.HasSuffix(strings.ToLower(fileName), ".xeh") {
		fileName += ".xeh"
	}

	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("[Error] すでに存在します: %s\n", fileName)
		return true
	}

	if err := os.WriteFile(fileName, []byte(defaultXehTemplate()), 0644); err != nil {
		fmt.Printf("[Error] %s の作成に失敗しました: %v\n", fileName, err)
		return true
	}

	fmt.Printf("新しい xeh ファイル %s を作成しました。\n", fileName)
	return true
}

// defaultSetJSON は `xeh new project` 用の最小限の set.json テンプレートを返す。
func defaultSetJSON(name string) string {
	return fmt.Sprintf(`{
  "meta": {
    "name": "%s",
    "version": "0.1.0",
    "license": "MIT",
    "charset": "UTF-8"
  },
  "handlers": {
    "xeh-logic": "core.logic"
  },
  "runtimes": {},
  "engines": {},
  "plugins": {}
}
`, name)
}

// defaultXehTemplate は `xeh new project` / `xeh new xehfile` 用の最小限の .xeh テンプレートを返す。
func defaultXehTemplate() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<xeh>
    <root memory_name="main_space">
        [
            {
                "name": "app_status",
                "type": "string",
                "value": "new"
            }
        ]
    </root>

    <subframework roteapp="/sub">
        <xeh-logic>
            <print value="Hello from new xeh file!" />
        </xeh-logic>
    </subframework>
</xeh>
`
}

// defaultPluginTemplate は `xeh new plugin` 用の雛形を、拡張子に応じて返す。
func defaultPluginTemplate(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".py":
		return `import sys
import json

print("[Plugin] started.")

for line in sys.stdin:
    try:
        packet = json.loads(line.strip())
        print(f"[Plugin] received: {packet}")
    except Exception as e:
        print(f"[Plugin Error]: {e}")
        break
`
	case ".rb":
		return `# TODO: プラグインのロジックをここに実装する
puts "[Plugin] started."
`
	case ".go":
		return `package main

func main() {
	// TODO: プラグインのロジックをここに実装する
}
`
	default:
		return "// TODO: プラグインのロジックをここに実装する\n"
	}
}

// setupSignalHandler はCtrl+C等のシグナルを受け取り、起動済みの全子プロセスを
// クリーンに終了させてからプログラムを終了する。
func setupSignalHandler(processes *core.ProcessList) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n--- Terminating xeh engine. Cleaning up subprocesses. ---")
		processes.KillAll()
		os.Exit(0)
	}()
}
