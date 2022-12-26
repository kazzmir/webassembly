package main

import (
    "log"
    "os"
    "fmt"
    "path/filepath"
    "github.com/kazzmir/webassembly/lib/core"
)

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
                module, err := wast.CreateWasmModule()
                if err != nil {
                    log.Printf("Error: %v", err)
                } else {
                    fmt.Println(module.ConvertToWat(""))
                }
            }
        }
    } else {
        log.Printf("Give a webassembly file to run\n")
    }
}
