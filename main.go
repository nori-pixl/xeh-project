package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type XehNode struct {
	XMLName  xml.Name
	Value    string    `xml:"value,attr"`
	Content  string    `xml:",chardata"`
	Children []XehNode `xml:",any"`
}

type SubFramework struct {
	RoteApp   string    `xml:"roteapp,attr"`
	ImportKey string    `xml:"import,attr"`
	Nodes     []XehNode `xml:",any"`
}

type XehApp struct {
	XMLName       xml.Name `xml:"xeh"`
	MemoryName    string   `xml:"root>memory_name,attr"`
	RootVariables string   `xml:"root"`
	SubFramework  []SubFramework `xml:"subframework"`
}

type XehVariable struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type MemoryPacket struct {
	MemoryName string                 `json:"memory_name"`
	Data       map[string]interface{} `json:"data"`
}

type MetaSetting struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	Charset string `json:"charset"`
}

type RuntimeSetting struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type EngineSetting struct {
	Type string `json:"type"`
	Src  string `json:"src"`
}

type PluginSetting struct {
	Src string `json:"src"`
}

type SetConfig struct {
	Meta     MetaSetting               `json:"meta"`
	Runtimes map[string]RuntimeSetting `json:"runtimes"`
	Engines  map[string]EngineSetting  `json:"engines"`
	Plugins  map[string]PluginSetting  `json:"plugins"`
}

var activeProcesses []*os.Process
var sharedStore = make(map[string]interface{})
var dynamicEngineMap = make(map[string]EngineSetting)

func main() {
	setFile, err := os.Open("set.json")
	if err != nil {
		log.Fatalf("[Error] Failed to open set.json: %v", err)
	}
	setBytes, _ := io.ReadAll(setFile)
	var setConfig SetConfig
	if err := json.Unmarshal(setBytes, &setConfig); err != nil {
		log.Fatalf("[Error] Failed to parse set.json: %v", err)
	}
	setFile.Close()

	for k, v := range setConfig.Engines {
		dynamicEngineMap[k] = v
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Printf("%s version %s\n", setConfig.Meta.Name, setConfig.Meta.Version)
			return
		}
		if os.Args[1] == "--license" {
			fmt.Printf("%s is licensed under the %s License.\n", setConfig.Meta.Name, setConfig.Meta.License)
			return
		}
	}

	fmt.Printf("--- %s (Version %s / %s License / %s) ---\n", 
		setConfig.Meta.Name, setConfig.Meta.Version, setConfig.Meta.License, setConfig.Meta.Charset)

	file, err := os.Open("app.xeh")
	if err != nil {
		log.Fatalf("[Error] Failed to open app.xeh: %v", err)
	}
	xmlBytes, _ := io.ReadAll(file)
	var mainApp XehApp
	xml.Unmarshal(xmlBytes, &mainApp)
	file.Close()

	memSpaceName := mainApp.MemoryName
	if memSpaceName == "" {
		memSpaceName = "default_space"
	}

	rootJSON := strings.TrimSpace(mainApp.RootVariables)
	if rootJSON != "" {
		var rawVariables []XehVariable
		if err := json.Unmarshal([]byte(rootJSON), &rawVariables); err != nil {
			log.Fatalf("[Error] Failed to parse root JSON: %v", err)
		}
		
		fmt.Printf("[xeh/os] Allocating memory space: '%s'\n", memSpaceName)
		for _, v := range rawVariables {
			sharedStore[v.Name] = v.Value
			fmt.Printf("   -> %s = %v\n", v.Name, v.Value)
		}
	}

	setupSignalHandler()

	for _, sub := range mainApp.SubFramework {
		if sub.ImportKey != "" {
			pluginConfig, exists := setConfig.Plugins[sub.ImportKey]
			if !exists {
				log.Printf("[Warning] Plugin key '%s' is not defined in set.json.", sub.ImportKey)
			} else {
				importedApp := loadXehFile(pluginConfig.Src)
				mergeRootVariables(importedApp.RootVariables)
			}
		}

		for _, node := range sub.Nodes {
			tagName := node.XMLName.Local

			if engConfig, assigned := dynamicEngineMap[tagName]; assigned {
				langSet, exists := setConfig.Runtimes[engConfig.Type]
				if !exists {
					log.Printf("[Error] Target runtime '%s' for engine '<%s>' is not defined in set.json.", engConfig.Type, tagName)
					continue
				}

				dir := filepath.Dir(engConfig.Src)
				if dir != "." {
					os.MkdirAll(dir, 0755)
				}
				
				f, _ := os.OpenFile(engConfig.Src, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				f.WriteString(strings.TrimSpace(node.Content))
				f.Close()

				finalArgs := make([]string, len(langSet.Args))
				for i, arg := range langSet.Args {
					finalArgs[i] = strings.ReplaceAll(arg, "{src}", engConfig.Src)
				}

				// Non-blocking asynchronous execution with input stream boundary termination
				go func() {
					var cmd *exec.Cmd
					rawArgs := strings.Join(finalArgs, " ")

					if os.PathSeparator == '\\' {
						cmd = exec.Command("cmd", "/C", langSet.Command+" "+rawArgs)
					} else {
						cmd = exec.Command("sh", "-c", langSet.Command+" "+rawArgs)
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
					activeProcesses = append(activeProcesses, cmd.Process)

					packet := MemoryPacket{
						MemoryName: memSpaceName,
						Data:       sharedStore,
					}
					packetBytes, _ := json.Marshal(packet)
					io.WriteString(stdinPipe, string(packetBytes)+"\n")
					
					// Close write pipeline explicitly to release subprocess scanner blocks
					stdinPipe.Close()

					cmd.Wait()
				}()
			}

			if tagName == "xeh-logic" {
				fmt.Println("[xeh/os] Executing native xeh XML logic:")
				for _, child := range node.Children {
					if child.XMLName.Local == "print" {
						output := child.Value
						if output == "" {
							output = child.Content
						}
						for k, v := range sharedStore {
							output = strings.ReplaceAll(output, "${"+k+"}", fmt.Sprintf("%v", v))
						}
						fmt.Printf("   [xeh Print]: %s\n", strings.TrimSpace(output))
					}
				}
			}
		}
	}

	select {}
}

func loadXehFile(filePath string) XehApp {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[Error] Failed to read plugin file: %s, %v", filePath, err)
		return XehApp{}
	}
	defer file.Close()
	bytes, _ := io.ReadAll(file)
	var app XehApp
	xml.Unmarshal(bytes, &app)
	return app
}

func mergeRootVariables(jsonStr string) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return
	}
	var tempVariables []XehVariable
	if err := json.Unmarshal([]byte(jsonStr), &tempVariables); err == nil {
		for _, v := range tempVariables {
			sharedStore[v.Name] = v.Value
		}
	}
}

func etupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n--- Terminating xeh engine. Cleaning up subprocesses. ---")
		for _, proc := range activeProcesses {
			if proc != nil {
				proc.Kill()
			}
		}
		os.Exit(0)
	}()
}
