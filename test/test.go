package main

import (
    "fmt"
    _ "io"
    "os"
    "strings"
    "path/filepath"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/sexp"
    "github.com/kazzmir/webassembly/lib/exec"
)

func compareSExpression(s1 sexp.SExpression, s2 sexp.SExpression) bool {
    s1.SortOne()
    s2.SortOne()
    // fmt.Printf("Compare:\n%v\n%v\n", s1, s2)
    return s1.Equal(&s2)
}

func compare(wasmPath string, expectedWatPath string) error {
    module, err := core.ParseWasmFile(wasmPath, false)
    if err != nil {
        return err
    }

    output := module.ConvertToWat("")

    // fmt.Printf("Output: %v\n", output)

    sexprActual, err := sexp.ParseSExpression(output)
    if err != nil {
        return err
    }

    // fmt.Printf("Read s-expr %v\n", sexprActual.String())

    data, err := os.ReadFile(expectedWatPath)
    if err != nil {
        return err
    }
    sexprExpected, err := sexp.ParseSExpression(string(data))
    if err != nil {
        return err
    }

    if !compareSExpression(sexprActual, sexprExpected) {
        return fmt.Errorf("sexpressions differed:\n%v\n%v", sexprActual, sexprExpected)
    }

    return nil
}

func ReplaceExtension(path string, newExt string) string {
    oldExt := filepath.Ext(path)
    base := strings.TrimSuffix(path, oldExt)
    return base + newExt
}

func runWasmToWatTests(){
    fmt.Printf("* .wasm -> .wat tests:\n")
    paths, err := os.ReadDir("test-files")
    if err != nil {
        fmt.Printf("Error: could not read test-files directory: %v\n", err)
        return
    }

    for _, path := range paths {
        name := path.Name()
        if !path.IsDir() && strings.HasSuffix(name, ".wasm"){
            wasm := filepath.Join("test-files", name)
            expectedWat := filepath.Join("test-files", ReplaceExtension(name, ".wat"))

            err := compare(wasm, expectedWat)
            if err != nil {
                fmt.Printf("Failure: %v vs %v: %v\n", wasm, expectedWat, err)
            } else {
                fmt.Printf("Success: %v\n", name)
            }
        }
    }
}

func runWastFile(path string) error {
    wast, err := core.ParseWastFile(path)
    if err != nil {
        return err
    }

    var module core.WebAssemblyModule
    var store *exec.Store

    for _, command := range wast.Expressions {
        switch command.Name {
            case "module":
                var err error
                module, err = core.CreateWasmModule(&command)
                if err != nil {
                    return err
                }
                store = exec.InitializeStore(module)
            case "assert_return":
                if store == nil {
                    return fmt.Errorf("Error: no module defined")
                }

                err := exec.AssertReturn(module, command, store)
                if err != nil {
                    return err
                }
        }
    }

    return nil
}

func runWastExecTests(){
    fmt.Printf("* Run .wast files:\n")
    paths, err := os.ReadDir("test-files/wast/")
    if err != nil {
        fmt.Printf("Error: could not read test-files directory: %v\n", err)
        return
    }

    for _, path := range paths {
        name := path.Name()
        if !path.IsDir() && strings.HasSuffix(name, ".wast"){
            fullPath := filepath.Join("test-files/wast", name)

            err := runWastFile(fullPath)
            if err != nil {
                fmt.Printf("Failure: %v: %v\n", name, err)
            } else {
                fmt.Printf("Success: %v\n", name)
            }
        }
    }
}

func runTests(){
    runWasmToWatTests()
    runWastExecTests()
}

func main() {
    runTests()
}
