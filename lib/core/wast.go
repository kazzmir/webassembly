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

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func MakeFunctionType(function *sexp.SExpression) WebAssemblyFunction {
    var out WebAssemblyFunction

    // (func $name (param ...) (result ...) code ...)
    // param or result might not exist, but if they do exist they will appear in positions 1/2
    for i := 1; i < min(3, len(function.Children)); i++ {
        if function.Children[i].Name == "param" {
            out.InputTypes = ConvertValueTypes(function.Children[i])
        } else if function.Children[i].Name == "result" {
            out.OutputTypes = ConvertValueTypes(function.Children[i])
        }
    }

    return out
}

func MakeExpressions(module WebAssemblyModule, expr *sexp.SExpression) []Expression {
    switch expr.Name {
        case "block", "loop":
            var children []Expression
            var expectedType []ValueType
            for _, child := range expr.Children {
                if child.Name == "result" {
                    for _, result := range child.Children {
                        expectedType = append(expectedType, ValueTypeFromName(result.Value))
                    }
                    continue
                }
                children = append(children, MakeExpressions(module, child)...)
            }

            var kind BlockKind = BlockKindBlock
            if expr.Name == "loop" {
                kind = BlockKindLoop
            }

            return []Expression{&BlockExpression{
                    Instructions: children,
                    Kind: kind,
                    ExpectedType: expectedType,
                },
            }
        case "if":
            var out []Expression
            var expectedType []ValueType

            var thenInstructions []Expression
            var elseInstructions []Expression

            for _, child := range expr.Children {
                if child.Name == "result" {
                    for _, result := range child.Children {
                        expectedType = append(expectedType, ValueTypeFromName(result.Value))
                    }
                    continue
                }

                if child.Name == "then" {
                    for _, then := range child.Children {
                        thenInstructions = append(thenInstructions, MakeExpressions(module, then)...)
                    }
                } else if child.Name == "else" {
                    for _, expr := range child.Children {
                        elseInstructions = append(elseInstructions, MakeExpressions(module, expr)...)
                    }
                } else {
                    out = append(out, MakeExpressions(module, child)...)
                }
            }

            return append(out, &BlockExpression{
                    Instructions: thenInstructions,
                    ElseInstructions: elseInstructions,
                    Kind: BlockKindIf,
                    ExpectedType: expectedType,
                })
        case "select":
            var out []Expression

            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, child)...)
            }

            return append(out, &SelectExpression{})
        case "br":
            label, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            if len(expr.Children) > 1 {
                return append(MakeExpressions(module, expr.Children[1]), &BranchExpression{Label: uint32(label)})
            } else {
                return []Expression{&BranchExpression{Label: uint32(label)}}
            }

        case "br_if":
            label, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            out := MakeExpressions(module, expr.Children[1])

            if len(expr.Children) > 2 {
                out = append(out, MakeExpressions(module, expr.Children[2])...)
            }

            return append(out, &BranchIfExpression{Label: uint32(label)})
        case "br_table":
            var out []Expression
            var labels []uint32
            // (br_table l1 l2 l3 (expr ...) (expr ...))
            for _, child := range expr.Children {
                if child.Value != "" {
                    label, err := strconv.Atoi(child.Value)
                    if err != nil {
                        return nil
                    }

                    labels = append(labels, uint32(label))
                } else {
                    // FIXME: once we start seeing expressions we shouldn't see any more labels, try to enforce this
                    out = append(out, MakeExpressions(module, child)...)
                }
            }

            return append(out, &BranchTableExpression{Labels: labels})
        case "return":
            if len(expr.Children) > 0 {
                return append(MakeExpressions(module, expr.Children[0]), &ReturnExpression{})
            }
            return []Expression{&ReturnExpression{}}
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
            return append(MakeExpressions(module, expr.Children[0]), &I32CtzExpression{})
        case "i64.const":
            value, err := strconv.ParseInt(expr.Children[0].Value, 10, 64)
            if err != nil {
                return nil
            }

            return []Expression{
                &I64ConstExpression{
                    N: value,
                },
            }

        case "i64.ctz":
            return append(MakeExpressions(module, expr.Children[0]), &I64CtzExpression{})
        case "i32.add":
            arg1 := MakeExpressions(module, expr.Children[0])
            arg2 := MakeExpressions(module, expr.Children[1])
            out := append(arg1, arg2...)
            return append(out, &I32AddExpression{})
        case "call_indirect":
            var typeIndex *TypeIndex
            tableId := 0
            typeStart := 0

            if expr.Children[0].Value != "" {
                value, err := strconv.Atoi(expr.Children[0].Value)
                if err != nil {
                    return nil
                }

                tableId = value
                typeStart = 1
            }

            type_ := expr.Children[typeStart]
            typeIndex = module.GetTypeSection().GetTypeByName(type_.Children[0].Value)

            var out []Expression
            for _, child := range expr.Children[typeStart+1:] {
                out = append(out, MakeExpressions(module, child)...)
            }

            return append(out, &CallIndirectExpression{
                Index: typeIndex,
                Table: &TableIndex{Id: uint32(tableId)},
            })

        case "call":
            name := expr.Children[0].Value
            index, ok := module.GetFunctionSection().GetFunctionIndexByName(name)
            if !ok {
                fmt.Printf("Unknown function in call '%v'\n", name)
                return nil
            }

            var out []Expression
            for _, child := range expr.Children[1:] {
                out = append(out, MakeExpressions(module, child)...)
            }

            return append(out, &CallExpression{Index: &FunctionIndex{uint32(index)}})
        case "drop":
            argument := MakeExpressions(module, expr.Children[0])
            return append(argument, &DropExpression{})
        case "f32.const":
            value, err := strconv.ParseFloat(expr.Children[0].Value, 32)
            if err != nil {
                return nil
            }
            return []Expression{
                &F32ConstExpression{
                    N: float32(value),
                },
            }
        case "f64.const":
            value, err := strconv.ParseFloat(expr.Children[0].Value, 64)
            if err != nil {
                return nil
            }
            return []Expression{
                &F64ConstExpression{
                    N: value,
                },
            }
        case "f32.neg":
            if len(expr.Children) > 0 {
                return append(MakeExpressions(module, expr.Children[0]), &F32NegExpression{})
            }

            return []Expression{&F32NegExpression{}}
        case "f64.neg":
            if len(expr.Children) > 0 {
                return append(MakeExpressions(module, expr.Children[0]), &F64NegExpression{})
            }

            return []Expression{&F64NegExpression{}}
        case "local.get":
            index, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            return []Expression{&LocalGetExpression{Local: uint32(index)}}
        case "local.set":
            index, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                return nil
            }

            var out []Expression
            if len(expr.Children) > 1 {
                out = MakeExpressions(module, expr.Children[1])
            }

            return append(out, &LocalSetExpression{Local: uint32(index)})

    }

    fmt.Printf("Warning: unhandled wast expression '%v'\n", expr.Name)

    return nil
}

func MakeCode(module WebAssemblyModule, function *sexp.SExpression) Code {
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
            /* FIXME: compress equal locals. i32 i32 i32 -> count=3 */
            out.Locals = append(out.Locals, Local{
                Count: 1,
                Type: ValueTypeFromName(current.Children[0].Value),
            })
        } else {
            /* FIXME: will we ever have more than 1 expression in the body of a function? */
            expressions := MakeExpressions(module, current)
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
    typeSection := NewWebAssemblyTypeSection()
    functionSection := WebAssemblyFunctionSectionCreate()
    codeSection := new(WebAssemblyCodeSection)
    tableSection := new(WebAssemblyTableSection)
    exportSection := new(WebAssemblyExportSection)
    elementSection := new(WebAssemblyElementSection)

    moduleOut.AddSection(typeSection)
    moduleOut.AddSection(functionSection)
    moduleOut.AddSection(codeSection)
    moduleOut.AddSection(tableSection)
    moduleOut.AddSection(elementSection)
    moduleOut.AddSection(exportSection)

    for _, expr := range wast.Module.Children {
        /*
        if expr.Name == "func" {
            fmt.Printf("Func: %v\n", expr)
        }
        */

        switch expr.Name {
            case "func":
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
                    }, cleanName(functionName.Value))

                    code := MakeCode(moduleOut, expr)

                    codeSection.AddCode(code)

                    exportSection.AddExport(cleanName(functionName.Value), &FunctionIndex{Id: functionIndex})
                } else {
                    functionType := MakeFunctionType(expr)
                    typeIndex := typeSection.GetOrCreateFunctionType(functionType)

                    _ = functionSection.AddFunction(&TypeIndex{
                        Id: typeIndex,
                    }, name.Value)

                    codeSection.AddCode(MakeCode(moduleOut, expr))
                }
            case "type":
                name := expr.Children[0]
                kind := expr.Children[1]
                if kind.Name == "func" {
                    typeIndex := typeSection.GetOrCreateFunctionType(MakeFunctionType(kind))
                    typeSection.AssociateName(name.Value, &TypeIndex{Id: typeIndex})
                }
            case "table":
                // so far this handles an inline table expression with funcref elements already given
                reftype := expr.Children[0]
                if reftype.Value == "funcrec" {
                    elements := expr.Children[1]
                    if elements.Name == "elem" {
                        tableId := tableSection.AddTable(TableType{
                            Limit: Limit{
                                Minimum: uint32(len(elements.Children)),
                                Maximum: uint32(len(elements.Children)),
                                HasMaximum: true,
                            },
                            RefType: RefTypeFunction,
                        })

                        var functions []*FunctionIndex
                        for _, element := range elements.Children {
                            if element.Value != "" {
                                functionIndex, ok := functionSection.GetFunctionIndexByName(element.Value)
                                if ok {
                                    functions = append(functions, &FunctionIndex{Id: uint32(functionIndex)})
                                } else {
                                    fmt.Printf("Warning: unable to find funcref '%v'\n", element.Value)
                                }
                            }
                        }

                        elementSection.AddFunctionRefInit(functions, int(tableId), []Expression{&I32ConstExpression{N: 0}})
                    }
                }

            default:
                fmt.Printf("Warning: unhandled wast top level '%v'\n", expr.Name)
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
