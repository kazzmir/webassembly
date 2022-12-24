package main

import (
    "log"
    "os"
    "fmt"
    "github.com/kazzmir/webassembly/lib/core"
)

func main(){
    log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
    log.Printf("Web assembly runner\n")

    if len(os.Args) > 1 {
        module, err := core.Parse(os.Args[1])
        if err != nil {
            log.Printf("Error: %v\n", err)
        } else {
            fmt.Println(module.ConvertToWast(""))
        }
    } else {
        log.Printf("Give a webassembly file to run\n")
    }
}
