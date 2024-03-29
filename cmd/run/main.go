package main

import (
    "log"
    "os"
    "fmt"
    "strings"
    "path/filepath"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/exec"
)

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

func handleWast(wast core.Wast){

    var module core.WebAssemblyModule
    var store *exec.Store

    for _, command := range wast.Expressions {
        switch command.Name {
            case "module":
                var err error
                module, err = core.CreateWasmModule(&command)
                if err != nil {
                    log.Printf("Error creating module: %v", err)
                    return
                }
                store = exec.InitializeStore(module)
            case "assert_return":
                if store == nil {
                    fmt.Printf("Error: no module defined\n")
                    continue
                }

                fmt.Printf("Execute %v\n", command.String())
                err := exec.AssertReturn(module, command, store)
                if err != nil {
                    fmt.Printf("Error: %v\n", err)
                }
        }
    }
}

func main(){
    log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
    log.Printf("Web assembly runner\n")

    if len(os.Args) > 1 {
        path := os.Args[1]
        if filepath.Ext(path) == ".wasm" {
            module, err := core.ParseWasmFile(path, true)
            if err != nil {
                log.Printf("Error: %v\n", err)
            } else {
                fmt.Println(module.ConvertToWat(""))
            }
        } else if filepath.Ext(path) == ".wast" {
            wast, err := core.ParseWastFile(path)
            if err != nil {
                log.Printf("Error: %v\n", err)
            } else {
                handleWast(wast)
            }
        }
    } else {
        log.Printf("Give a webassembly file to run\n")
    }
}
