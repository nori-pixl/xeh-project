package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"xeh/internal/core"
)

func init() {
	core.Register("core.engine", RunEngine)
}

// RunEngine は set.json の "engines" 設定に基づき、タグの中身を
// 指定ファイルに書き出し、対応する "runtimes" コマンドで非同期実行する。
//
// set.json側では以下のように指定する:
//
//	"handlers": { "python": "core.engine" },
//	"engines":  { "python": { "type": "python", "src": "engines/main.py" } },
//	"runtimes": { "python": { "command": "python3", "args": ["{src}"] } }
func RunEngine(node core.XehNode, ctx *core.Context) error {
	tagName := ctx.TagName

	engConfig, ok := ctx.SetConfig.Engines[tagName]
	if !ok {
		return fmt.Errorf("no engine config for tag '<%s>' in set.json (engines)", tagName)
	}

	runtimeSet, ok := ctx.SetConfig.Runtimes[engConfig.Type]
	if !ok {
		return fmt.Errorf("runtime '%s' for engine '<%s>' is not defined in set.json (runtimes)", engConfig.Type, tagName)
	}

	if err := writeEngineSource(engConfig.Src, node.Content); err != nil {
		return fmt.Errorf("failed to write engine src for <%s>: %w", tagName, err)
	}

	finalArgs := buildArgs(runtimeSet.Args, engConfig.Src)

	ctx.WaitGroup.Add(1)
	go runEngineProcess(tagName, runtimeSet, finalArgs, ctx)

	return nil
}

// writeEngineSource はタグの中身を指定ファイルに書き出す
func writeEngineSource(src, content string) error {
	if dir := filepath.Dir(src); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", dir, err)
		}
	}

	f, err := os.OpenFile(src, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", src, err)
	}
	defer f.Close()

	if _, err := f.WriteString(strings.TrimSpace(content)); err != nil {
		return fmt.Errorf("failed to write file '%s': %w", src, err)
	}

	return nil
}

// buildArgs はRuntimeSetting.Args内の "{src}" を実ファイルパスに置換する
func buildArgs(args []string, src string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = strings.ReplaceAll(arg, "{src}", src)
	}
	return out
}

// runEngineProcess は外部プロセスを起動し、共有メモリのスナップショットをstdinで渡して実行する
func runEngineProcess(tagName string, runtimeSet core.RuntimeSetting, args []string, ctx *core.Context) {
	defer ctx.WaitGroup.Done()

	rawArgs := strings.Join(args, " ")

	var cmd *exec.Cmd
	if os.PathSeparator == '\\' {
		cmd = exec.Command("cmd", "/C", runtimeSet.Command+" "+rawArgs)
	} else {
		cmd = exec.Command("sh", "-c", runtimeSet.Command+" "+rawArgs)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("[Error] Failed to create stdin pipe for <%s>: %v", tagName, err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[Error] Failed to start process for <%s>: %v", tagName, err)
		return
	}

	ctx.Processes.Add(cmd.Process)

	packet := core.MemoryPacket{
		MemoryName: ctx.MemSpaceName,
		Data:       ctx.Store.Snapshot(),
	}
	packetBytes, err := json.Marshal(packet)
	if err != nil {
		log.Printf("[Error] Failed to marshal memory packet for <%s>: %v", tagName, err)
	} else {
		_, _ = io.WriteString(stdinPipe, string(packetBytes)+"\n")
	}
	stdinPipe.Close()

	if err := cmd.Wait(); err != nil {
		log.Printf("[Warning] Process for <%s> exited with error: %v", tagName, err)
	}
}
