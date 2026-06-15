package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"strings"

	"xeh/internal/core"
	"xeh/internal/handlers"
)

func main() {
	// set.json = 設計図。タグ名 -> ハンドラーID のマッピングなどを保持する。
	setConfig := core.LoadSetConfig("set.json")

	if handleCLIFlags(setConfig) {
		return
	}

	fmt.Printf("--- %s (Version %s / %s License / %s) ---\n",
		setConfig.Meta.Name, setConfig.Meta.Version, setConfig.Meta.License, setConfig.Meta.Charset)

	mainApp := core.MustLoadXehFile("app.xeh")

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

// handleCLIFlags は --version / --license などのCLIフラグを処理する。
// 処理した場合は true を返し、main はそのまま終了する。
func handleCLIFlags(cfg *core.SetConfig) bool {
	if len(os.Args) <= 1 {
		return false
	}

	switch strings.ToLower(os.Args[1]) {
	case "--version", "-v":
		fmt.Printf("%s version %s\n", cfg.Meta.Name, cfg.Meta.Version)
		return true
	case "--license":
		fmt.Printf("%s is licensed under the %s License.\n", cfg.Meta.Name, cfg.Meta.License)
		return true
	case "--help", "-h":
		fmt.Printf("--version, --v\n")
		fmt.Printf("--license\n")
		fmt.Printf("--help, -h\n")
		fmt.Printf("--mail\n")
		fmt.Printf("--config\n")
		return true
	case "--mail":
		fmt.Printf("gmail: sato.shigure4@gmail.com")
		return true
	case "--config":
		fmt.Printf("set.json")
		return true
	    // switchの最後に追加する
    default:
        fmt.Printf("No command available.": %s\n", os.Args[1])
        fmt.Println("To check how to use it, run -help.")
        return true
	}

	return false
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
