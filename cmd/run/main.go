package main

import (
    "log"
    "os"
    "fmt"
    "strings"
    "path/filepath"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/sexp"
    "github.com/kazzmir/webassembly/lib/exec"
)

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

func doAssertReturn(module core.WebAssemblyModule, assert sexp.SExpression){
    what := assert.Children[0]
    if what.Name == "invoke" {
        // FIXME: add args
        result, err := exec.Invoke(module, cleanName(what.Children[0].Value))
        if err != nil {
            fmt.Printf("Error: %v\n", err)
        } else {
            fmt.Printf("Result: %v\n", result)
        }
    }
}

func handleWast(wast core.Wast){
    module, err := wast.CreateWasmModule()
    if err != nil {
        log.Printf("Error: %v", err)
        return
    } else {
        fmt.Println(module.ConvertToWat(""))
    }

    for _, command := range wast.Expressions {
        if command.Name == "assert_return" {
            doAssertReturn(module, command)
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
