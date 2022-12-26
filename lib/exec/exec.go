package exec

import (
    "fmt"
    "github.com/kazzmir/webassembly/lib/core"
)

type RuntimeValue struct {
}

func Invoke(module core.WebAssemblyModule, function string) (RuntimeValue, error) {
    kind := module.GetExportSection().FindExportByName(function)
    if kind == nil {
        return RuntimeValue{}, fmt.Errorf("no such exported function '%v'", function)
    }
   
    return RuntimeValue{}, nil
}
