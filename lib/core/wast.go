package core

import (
    "bufio"
    "os"
    "io"
    "errors"
    "fmt"
    "strconv"
    "strings"
    "github.com/kazzmir/webassembly/lib/sexp"
)

// wast is a super set of .wat in that it can contain (module ...) expressions as well as other things
// like (assert_return ...) and other things
type Wast struct {
    Module sexp.SExpression
    Expressions []sexp.SExpression
}

func ConvertValueTypes(expr *sexp.SExpression) []ValueType {
    var out []ValueType

    for _, child := range expr.Children {
        out = append(out, ValueTypeFromName(child.Value))
    }

    return out
}

func MakeFunctionType(function *sexp.SExpression) WebAssemblyFunction {
    var out WebAssemblyFunction

    switch len(function.Children) {
        // (func name (params ...) code)
        // (func name (result ...) code)
        case 3:
            if function.Children[2].Name == "params" {
                out.InputTypes = ConvertValueTypes(function.Children[2])
            } else if function.Children[2].Name == "result" {
                out.OutputTypes = ConvertValueTypes(function.Children[2])
            }
        // (func name (params ...) (result ...) code)
        case 4:
            if function.Children[2].Name == "params" {
                out.InputTypes = ConvertValueTypes(function.Children[2])
            }

            if function.Children[3].Name == "result" {
                out.OutputTypes = ConvertValueTypes(function.Children[3])
            }
    }

    return out
}

func MakeExpressions(expr *sexp.SExpression) []Expression {
    switch expr.Name {
        case "block":
            var children []Expression
            for _, child := range expr.Children {
                if child.Name == "result" {
                    continue
                }
                children = append(children, MakeExpressions(child)...)
            }

            return []Expression{&BlockExpression{
                    Instructions: children,
                    Kind: BlockKindBlock,
                },
            }
        case "br_if":
            label, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            out := append(MakeExpressions(expr.Children[1]), MakeExpressions(expr.Children[2])...)

            return append(out, &BranchIfExpression{Label: uint32(label)})
        case "i32.const":
            value, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            return []Expression{
                &I32ConstExpression{
                    N: int32(value),
                },
            }
        case "i32.ctz":
            return append(MakeExpressions(expr.Children[0]), &I32CtzExpression{})
        case "i64.const":
            value, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            return []Expression{
                &I64ConstExpression{
                    N: int32(value),
                },
            }

        case "i64.ctz":
            return append(MakeExpressions(expr.Children[0]), &I64CtzExpression{})
        case "drop":
            argument := MakeExpressions(expr.Children[0])
            return append(argument, &DropExpression{})
    }

    fmt.Printf("Warning: unhandled expression '%v'\n", expr.Name)

    return nil
}

func MakeCode(function *sexp.SExpression) Code {
    var out Code

    startIndex := 1
    for {
        if startIndex < len(function.Children) {
            if function.Children[startIndex].Name == "param" || function.Children[startIndex].Name == "result" {
                startIndex += 1
            } else {
                break
            }
        } else {
            break
        }
    }

    for startIndex := startIndex; startIndex < len(function.Children); startIndex++ {
        current := function.Children[startIndex]
        if current.Name == "local" {
            out.Locals = append(out.Locals, Local{
                Count: 1,
                Type: ValueTypeFromName(current.Children[0].Value),
            })
        } else {
            expressions := MakeExpressions(current)
            out.Expressions = append(out.Expressions, expressions...)
        }
    }

    return out
}

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

func (wast *Wast) CreateWasmModule() (WebAssemblyModule, error) {
    if wast.Module.Name != "module" {
        return WebAssemblyModule{}, fmt.Errorf("No module defined")
    }

    var moduleOut WebAssemblyModule
    typeSection := new(WebAssemblyTypeSection)
    functionSection := new(WebAssemblyFunctionSection)
    codeSection := new(WebAssemblyCodeSection)
    exportSection := new(WebAssemblyExportSection)

    moduleOut.AddSection(typeSection)
    moduleOut.AddSection(functionSection)
    moduleOut.AddSection(codeSection)
    moduleOut.AddSection(exportSection)

    for _, expr := range wast.Module.Children {
        /*
        if expr.Name == "func" {
            fmt.Printf("Func: %v\n", expr)
        }
        */

        name := expr.Children[0]
        if name.Name == "export" {
            // 1. create a type that matches the given function and call typesection.AddFunctionType()
            // 2. create a function that references the newly created type and call functionsection.AddFunction()
            // 3. create a FunctionIndex and add it to the export section with exportSection.AddExport()
            functionName := name.Children[0]

            functionType := MakeFunctionType(expr)
            typeIndex := typeSection.GetOrCreateFunctionType(functionType)

            functionIndex := functionSection.AddFunction(&TypeIndex{
                Id: typeIndex,
            })

            code := MakeCode(expr)

            codeSection.AddCode(code)

            exportSection.AddExport(cleanName(functionName.Value), &FunctionIndex{Id: functionIndex})
        } else {
            functionType := MakeFunctionType(expr)
            typeIndex := typeSection.GetOrCreateFunctionType(functionType)

            _ = functionSection.AddFunction(&TypeIndex{
                Id: typeIndex,
            })

            var code Code
            codeSection.AddCode(code)
        }
    }

    return moduleOut, nil
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
