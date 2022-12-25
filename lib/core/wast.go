package core

import (
    "bufio"
    "os"
    "io"
    "errors"
    "github.com/kazzmir/webassembly/lib/sexp"
)

// wast is a super set of .wat in that it can contain (module ...) expressions as well as other things
// like (assert_return ...) and other things
type Wast struct {
    Module sexp.SExpression
    Expressions []sexp.SExpression
}

func ParseWastFile(path string) (Wast, error) {
    var wast Wast

    file, err := os.Open(path)
    if err != nil {
        return Wast{}, err
    }
    defer file.Close()

    reader := bufio.NewReader(file)

    for {
        next, err := sexp.ParseSExpressionReader(reader)
        if err != nil {
            if errors.Is(err, io.EOF) {
                break
            }
            return Wast{}, err
        }
        
        if next.Name == "module" {
            wast.Module = next
        } else {
            wast.Expressions = append(wast.Expressions, next)
        }
    }

    return wast, nil
}
