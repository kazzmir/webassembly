package core

import (
    "bufio"
    "os"
    "io"
    "errors"
    "fmt"
    "math"
    "strconv"
    "strings"
    // "regexp"
    "github.com/kazzmir/webassembly/lib/sexp"
    "github.com/kazzmir/webassembly/lib/data"
)

// wast is a super set of .wat in that it can contain (module ...) expressions as well as other things
// like (assert_return ...) and other things
type Wast struct {
    Expressions []sexp.SExpression
}

type SecondPassFunction func() Expression

type SecondPassExpression struct {
    Replace SecondPassFunction
}

func (expr *SecondPassExpression) ConvertToWat(x data.Stack[int], y string) string {
    return "second-pass"
}

func ConvertValueTypes(expr *sexp.SExpression) []ValueType {
    var out []ValueType

    for _, child := range expr.Children {
        value := ValueTypeFromName(child.Value)
        if value != InvalidValueType {
            out = append(out, value)
        }
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
    for i := 0; i < len(function.Children); i++ {
        if i == 0 && function.Children[i].Value != "" {
            continue
        }
        if function.Children[i].Name == "param" {
            param := function.Children[i]
            name := ""
            for z := 0; z < len(param.Children); z++ {
                if z == 0 && ValueTypeFromName(param.Children[i].Value) == InvalidValueType {
                    name = param.Children[i].Value
                    continue
                }
                if z > 0 {
                    name = ""
                }
                out.InputTypes = append(out.InputTypes, Parameter{
                    Type: ValueTypeFromName(param.Children[i].Value),
                    Name: name,
                })
            }
        } else if function.Children[i].Name == "result" {
            out.OutputTypes = ConvertValueTypes(function.Children[i])
        }
    }

    return out
}

/* try to parse as base 10 and also as base 16 */
func parseLiteralI32(data string) (int32, error) {
    x, err := strconv.ParseInt(data, 0, 32)
    if err == nil {
        return int32(x), nil
    }

    m, err := strconv.ParseUint(data, 0, 32)
    if err == nil {
        return int32(m), nil
    }

    // handle 0_123_456_789
    // remove _ and leading 0
    normalized := strings.TrimLeft(strings.ReplaceAll(data, "_", ""), "0")
    m, err = strconv.ParseUint(normalized, 0, 32)
    if err == nil {
        return int32(m), nil
    }

    return 0, err
}

func parseNum(reader *strings.Reader) (int64, error) {
    var out int64

    for {
        digit, err := reader.ReadByte()
        if err != nil {
            return out, nil
        }

        value, err := strconv.ParseInt(string(digit), 0, 64)
        if err != nil {
            reader.UnreadByte()
            return out, nil
        }

        out = out * 10 + value
    }

    return out, nil
}

func parseHexFloat(reader *strings.Reader) (float64, error) {
    var out float64

    for {
        digit, err := reader.ReadByte()
        if err != nil {
            return out, nil
        }

        value, err := strconv.ParseInt("0x" + string(digit), 0, 64)
        if err != nil {
            reader.UnreadByte()
            return out, nil
        }

        out = out * 16 + float64(value)
    }

    return out, nil
}

func parseHexFrac(reader *strings.Reader) (float64, error) {
    digit, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    value, err := strconv.ParseInt("0x" + string(digit), 0, 64)
    if err != nil {
        reader.UnreadByte()
        return 0, err
    }

    rest, err := parseHexFrac(reader)
    if err != nil {
        return float64(value) / 16, nil
    } else {
        return (float64(value) + rest/16) / 16, nil
    }
}

func parseRawFloat(reader *strings.Reader) (float64, error) {
    var out float64

    for {
        digit, err := reader.ReadByte()
        if err != nil {
            return out, nil
        }

        value, err := strconv.ParseInt(string(digit), 0, 64)
        if err != nil {
            reader.UnreadByte()
            return out, nil
        }

        out = out * 10 + float64(value)
    }

    return out, nil
}

/* parse a base-10 float */
    /*
func parseRawFloat(data string) (float64, error) {
    if strings.HasPrefix(data, "0x") {
        value, err = parseRawHexFloat(data)
        if err == nil {
            return value, nil
        }
    }

    var out float64

    for _, digit := range data {
        value, err := strconv.ParseInt(string(digit), 0, 64)
        if err != nil {
            return 0, err
        }

        out = out * 10 + float64(value)
    }

    return out, nil

    return 0, nil
}
    */

/* 0x <hexnum> .? */
func parseFloat64_1(reader *strings.Reader) (float64, error) {
    p, err := parseHexFloat(reader)
    if err != nil {
        return 0, err
    }

    if reader.Len() == 0 {
        return p, nil
    }

    next, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if next != '.' {
        return 0, fmt.Errorf("expected .")
    }

    if reader.Len() != 0 {
        return 0, fmt.Errorf("fail")
    }

    return p, nil
}

/* 0x <hexnum> . <hexfrac> */
func parseFloat64_2(reader *strings.Reader) (float64, error) {
    p, err := parseHexFloat(reader)
    if err != nil {
        return 0, err
    }

    next, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if next != '.' {
        return 0, fmt.Errorf("expected .")
    }

    q, err := parseHexFrac(reader)
    if err != nil {
        return 0, err
    }

    if reader.Len() != 0 {
        return 0, fmt.Errorf("fail")
    }

    return p+q, nil
}

/* 0x <hexnum> . p +- <num> */
func parseFloat64_3(reader *strings.Reader) (float64, error) {
    p, err := parseHexFloat(reader)
    if err != nil {
        return 0, err
    }

    next, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if next != '.' {
        reader.UnreadByte()
    }

    pChar, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if strings.ToLower(string(pChar)) != "p" {
        return 0, err
    }

    sign := 1
    next, err = reader.ReadByte()
    if err != nil {
        return 0, fmt.Errorf("fail")
    }

    if next == '-' {
        sign = -1
    } else if next == '+' {
        sign = 1
    } else {
        reader.UnreadByte()
    }

    e, err := parseNum(reader)
    if err != nil {
        return 0, err
    }

    if reader.Len() != 0 {
        return 0, fmt.Errorf("fail")
    }

    return p * math.Pow(2, float64(e) * float64(sign)), nil
}

/* 0x <hexnum> . <hexfrac> p +- <num> */
func parseFloat64_4(reader *strings.Reader) (float64, error) {
    p, err := parseHexFloat(reader)
    if err != nil {
        return 0, err
    }

    next, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if next != '.' {
        return 0, fmt.Errorf("expected .")
    }

    q, err := parseHexFrac(reader)
    if err != nil {
        return 0, err
    }

    pChar, err := reader.ReadByte()
    if err != nil {
        return 0, err
    }

    if strings.ToLower(string(pChar)) != "p" {
        return 0, err
    }

    sign := 1
    next, err = reader.ReadByte()
    if err != nil {
        return 0, fmt.Errorf("fail")
    }

    if next == '-' {
        sign = -1
    } else if next == '+' {
        sign = 1
    } else {
        reader.UnreadByte()
    }

    e, err := parseNum(reader)
    if err != nil {
        return 0, err
    }

    if reader.Len() != 0 {
        return 0, fmt.Errorf("fail")
    }

    return (p+q) * math.Pow(2, float64(e) * float64(sign)), nil
}

func parseFloat64(data string) (float64, error) {

    var sign float64 = 1
    if len(data) > 0 {
        if data[0] == '-' {
            sign = -1
            data = data[1:]
        } else if data[0] == '+' {
            sign = 1
            data = data[1:]
        }
    }

    if strings.HasPrefix(data, "nan") {
        // FIXME: handle payload
        return math.NaN(), nil
    }

    normalized := strings.ReplaceAll(data, "_", "")

    /* give the native go parser a whack at it */
    check, err := strconv.ParseFloat(normalized, 64)
    if err == nil {
        return check*sign, nil
    }

    if strings.HasPrefix(normalized, "0x") {
        left, err := parseFloat64_1(strings.NewReader(normalized[2:]))
        if err == nil {
            return sign*left, nil
        }

        left, err = parseFloat64_2(strings.NewReader(normalized[2:]))
        if err == nil {
            return sign*left, nil
        }

        left, err = parseFloat64_3(strings.NewReader(normalized[2:]))
        if err == nil {
            return sign*left, nil
        }

        left, err = parseFloat64_4(strings.NewReader(normalized[2:]))
        if err == nil {
            return sign*left, nil
        }

        return 0, fmt.Errorf("could not read float")
    } else {
        value, err := strconv.ParseFloat(normalized, 64)
        return sign*value, err
    }
}

func parseFloat32(data string) (float32, error) {
    x, err := parseFloat64(data)
    return float32(x), err
}

func MakeExpressions(module WebAssemblyModule, code *Code, labels data.Stack[string], expr *sexp.SExpression) []Expression {

    /* convert everything in the given sexp to expression sequences and append them all together */
    subexpressions := func(expr *sexp.SExpression) []Expression {
        var out []Expression
        for _, child := range expr.Children {
            out = append(out, MakeExpressions(module, code, labels, child)...)
        }
        return out
    }

    parseLabel := func(name string) (int, error) {
        label, err := strconv.Atoi(name)
        if err != nil {
            index, ok := labels.Find(name)
            if ok {
                return index, nil
            }

            return 0, err
        }

        return label, nil
    }

    if expr.Value != "" {
        /* handle flat syntax */
        switch expr.Value {
            case "drop":  return []Expression{&DropExpression{}}
        }
    }

    switch expr.Name {
        case "block", "loop":
            var children []Expression
            var expectedType []ValueType
            for i, child := range expr.Children {
                if child.Name == "result" {
                    for _, result := range child.Children {
                        expectedType = append(expectedType, ValueTypeFromName(result.Value))
                    }
                    continue
                }
                if child.Name == "param" {
                    /* FIXME: handle this */
                    continue
                }
                if child.Name == "type" {
                    /* FIXME: handle this */
                    continue
                }
                /* (block $x ...) */
                if i == 0 {
                    if child.Value != "" {
                        labels.Push(child.Value)
                        defer labels.Pop()
                        continue
                    } else {
                        labels.Push("") // push unnamed label
                        defer labels.Pop()
                    }
                }
                children = append(children, MakeExpressions(module, code, labels, child)...)
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

            labels.Push("") // unnamed label for if
            defer labels.Pop()

            for i, child := range expr.Children {
                if i == 0 && child.Value != "" {
                    labels.Push(child.Value)
                    defer labels.Pop()
                    continue
                }

                if child.Name == "param" {
                    // FIXME:
                    continue
                }
                if child.Name == "type" {
                    name := child.Children[0].Value
                    index := module.GetTypeSection().GetTypeByName(name)
                    if index != nil {
                        functionType := module.GetTypeSection().GetFunction(index.Id)
                        expectedType = functionType.OutputTypes
                    }
                    continue
                }
                if child.Name == "result" {
                    for _, result := range child.Children {
                        expectedType = append(expectedType, ValueTypeFromName(result.Value))
                    }
                    continue
                }

                if child.Name == "then" {
                    for _, then := range child.Children {
                        thenInstructions = append(thenInstructions, MakeExpressions(module, code, labels, then)...)
                    }
                } else if child.Name == "else" {
                    for _, expr := range child.Children {
                        elseInstructions = append(elseInstructions, MakeExpressions(module, code, labels, expr)...)
                    }
                } else {
                    out = append(out, MakeExpressions(module, code, labels, child)...)
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
                if child.Name == "result" {
                    // FIXME: handle this
                    continue
                }
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &SelectExpression{})
        case "br":
            var out []Expression

            for _, child := range expr.Children[1:] {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            label, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {

                index, ok := labels.Find(expr.Children[0].Value)
                if ok {
                    return append(out, &BranchExpression{Label: uint32(index)})
                }

                return nil
            }

            return append(out, &BranchExpression{Label: uint32(label)})
        case "nop":
            /* FIXME: does this need an actual expression object? */
            return nil
        case "br_if":
            var out []Expression

            for _, child := range expr.Children[1:] {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            label, err := parseLabel(expr.Children[0].Value)
            if err != nil {
                fmt.Printf("Error: could not parse label '%v': %v\n", expr.Children[0].Value, err)
                return nil
            }

            return append(out, &BranchIfExpression{Label: uint32(label)})
        case "br_table":
            var out []Expression
            var tableLabels []uint32
            // (br_table l1 l2 l3 (expr ...) (expr ...))
            for _, child := range expr.Children {
                if child.Value != "" {
                    label, err := parseLabel(child.Value)

                    if err != nil {
                        fmt.Printf("Error: could not parse label '%v': %v\n", child.Value, err)
                        return nil
                    }

                    tableLabels = append(tableLabels, uint32(label))
                } else {
                    // FIXME: once we start seeing expressions we shouldn't see any more labels, try to enforce this
                    out = append(out, MakeExpressions(module, code, labels, child)...)
                }
            }

            return append(out, &BranchTableExpression{Labels: tableLabels})
        case "return":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &ReturnExpression{})
        case "i32.const":
            value, err := parseLiteralI32(expr.Children[0].Value)
            if err != nil {
                fmt.Printf("Warning: could not parse i32.const literal '%v'\n", expr.Children[0].Value)
                return nil
            }

            return []Expression{
                &I32ConstExpression{
                    N: int32(value),
                },
            }
        case "i32.lt_u":
            return append(subexpressions(expr), &I32LtuExpression{})
        case "i32.lt_s":
            return append(subexpressions(expr), &I32LtsExpression{})
        case "i32.gt_u":
            return append(subexpressions(expr), &I32GtuExpression{})
        case "i32.gt_s":
            return append(subexpressions(expr), &I32GtsExpression{})
        case "i32.ge_s":
            return append(subexpressions(expr), &I32GesExpression{})
        case "i32.ge_u":
            return append(subexpressions(expr), &I32GeuExpression{})
        case "i32.eq":
            return append(subexpressions(expr), &I32EqExpression{})
        case "i32.ctz":
            return append(subexpressions(expr), &I32CtzExpression{})
        case "i32.clz":
            return append(subexpressions(expr), &I32ClzExpression{})
        case "i32.popcnt":
            return append(subexpressions(expr), &I32PopcntExpression{})
        case "i64.lt_s":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &I64LtsExpression{})
        case "i64.gt_s":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &I64GtsExpression{})
        case "i64.gt_u":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &I64GtuExpression{})
        case "i64.add":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &I64AddExpression{})
        case "i64.const":
            data := strings.ReplaceAll(expr.Children[0].Value, "_", "")
            var use int64
            value, err := strconv.ParseUint(data, 0, 64)
            if err == nil {
                use = int64(value)
            } else {
                value, err := strconv.ParseInt(data, 0, 64)
                if err == nil {
                    use = value
                } else {
                    fmt.Printf("Error: could not parse int64 constant: %v\n", err)
                    return nil
                }
            }

            return []Expression{
                &I64ConstExpression{
                    N: use,
                },
            }

        case "i64.ctz":
            return append(MakeExpressions(module, code, labels, expr.Children[0]), &I64CtzExpression{})
        case "i32.add":
            return append(subexpressions(expr), &I32AddExpression{})
        case "i32.mul":
            return append(subexpressions(expr), &I32MulExpression{})
        case "i64.sub":
            return append(subexpressions(expr), &I64SubExpression{})
        case "i32.sub":
            return append(subexpressions(expr), &I32SubExpression{})
        case "i64.eq":
            return append(subexpressions(expr), &I64EqExpression{})
        case "i32.eqz":
            return append(subexpressions(expr), &I32EqzExpression{})
        case "i32.le_u":
            return append(subexpressions(expr), &I32LeuExpression{})
        case "i32.le_s":
            return append(subexpressions(expr), &I32LesExpression{})
        case "i64.le_u":
            return append(subexpressions(expr), &I64LeuExpression{})
        case "i32.ne":
            return append(subexpressions(expr), &I32NeExpression{})
        case "i64.mul":
            return append(subexpressions(expr), &I64MulExpression{})
        case "i64.eqz":
            return append(subexpressions(expr), &I64EqzExpression{})
        case "memory.grow":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }
            return append(out, &MemoryGrowExpression{})

        case "call_indirect":
            var typeIndex *TypeIndex
            tableId := 0
            typeStart := 0

            if expr.Children[0].Value != "" {
                value, err := strconv.Atoi(expr.Children[0].Value)
                if err == nil {
                    tableId = value
                } else {
                    value, ok := module.GetTableSection().FindTableIndexByName(expr.Children[0].Value)
                    if !ok {
                        fmt.Printf("Error: could not find table '%v'\n", expr.Children[0].Value)
                        return nil
                    }
                    tableId = int(value)
                }

                typeStart = 1
            }

            type_ := expr.Children[typeStart]
            typeIndex = module.GetTypeSection().GetTypeByName(type_.Children[0].Value)

            var out []Expression
            for _, child := range expr.Children[typeStart+1:] {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            return append(out, &CallIndirectExpression{
                Index: typeIndex,
                Table: &TableIndex{Id: uint32(tableId)},
            })

        case "call":
            var out []Expression
            for _, child := range expr.Children[1:] {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            name := expr.Children[0].Value

            var index int
            value, err := strconv.Atoi(name)
            if err == nil {
                index = value
            } else {
                var ok bool
                /* look up the function by name, but if we can't find it now then the function might exist later
                 * once more functions are parsed. in case the function can't be found then insert a delayed
                 * expression that will get replaced in a second pass.
                 */
                index, ok = module.GetFunctionSection().GetFunctionIndexByName(name)
                if !ok {
                    return append(out, &SecondPassExpression{
                        Replace: func() Expression {
                            check, ok := module.GetFunctionSection().GetFunctionIndexByName(name)
                            if ok {
                                return &CallExpression{Index: &FunctionIndex{uint32(check)}}
                            } else {
                                fmt.Printf("Error: unknown function with name '%v'\n", name)
                                return nil
                            }
                        },
                    })
                    return nil
                }
            }

            return append(out, &CallExpression{Index: &FunctionIndex{uint32(index)}})
        case "unreachable":
            return append(subexpressions(expr), &UnreachableExpression{})
        case "ref.null":
            switch expr.Children[0].Value {
                case "func": return []Expression{&RefFuncNullExpression{}}
                case "externref": return []Expression{&RefExternNullExpression{}}
            }

            fmt.Printf("Error: invalid ref.null %v\n", expr.Children[0].Value)

            return nil
        case "ref.extern":
            value, err := parseLiteralI32(expr.Children[0].Value)
            if err != nil {
                fmt.Printf("Error: could not parse ref.extern index: %v", err)
                return nil
            }

            return []Expression{&RefExternExpression{Id: uint32(value)}}
        case "drop":
            return append(subexpressions(expr), &DropExpression{})
        case "i64.extend_i32_u":
            return append(subexpressions(expr), &I64ExtendI32uExpression{})
        case "i64.lt_u":
            return append(subexpressions(expr), &I64LtuExpression{})
        case "i32.div_s":
            return append(subexpressions(expr), &I32DivsExpression{})
        case "i32.div_u":
            return append(subexpressions(expr), &I32DivuExpression{})
        case "i32.rem_s":
            return append(subexpressions(expr), &I32RemsExpression{})
        case "i32.and":
            return append(subexpressions(expr), &I32AndExpression{})
        case "i32.or":
            return append(subexpressions(expr), &I32OrExpression{})
        case "i32.xor":
            return append(subexpressions(expr), &I32XOrExpression{})
        case "i32.shl":
            return append(subexpressions(expr), &I32ShlExpression{})
        case "i32.shl_s":
            return append(subexpressions(expr), &I32ShlsExpression{})
        case "i32.shl_u":
            return append(subexpressions(expr), &I32ShluExpression{})
        case "i32.shr_s":
            return append(subexpressions(expr), &I32ShrsExpression{})
        case "i32.shr_u":
            return append(subexpressions(expr), &I32ShruExpression{})
        case "i32.rem_u":
            return append(subexpressions(expr), &I32RemuExpression{})
        case "i32.rotl":
            return append(subexpressions(expr), &I32RotlExpression{})
        case "i32.rotr":
            return append(subexpressions(expr), &I32RotrExpression{})
        case "i32.extend8_s":
            return append(subexpressions(expr), &I32Extend8sExpression{})
        case "i32.extend16_s":
            return append(subexpressions(expr), &I32Extend16sExpression{})
        case "i64.extend_i32_s":
            return append(subexpressions(expr), &I64ExtendI32sExpression{})
        case "i64.trunc_f64_s":
            return append(subexpressions(expr), &I64TruncF64sExpression{})
        case "i32.wrap_i64":
            return append(subexpressions(expr), &I32WrapI64Expression{})
        case "f64.convert_i64_u":
            return append(subexpressions(expr), &F64ConvertI64uExpression{})
        case "f64.promote_f32":
            return append(subexpressions(expr), &F64PromoteF32Expression{})
        case "f64.convert_i32_u":
            return append(subexpressions(expr), &F64ConvertI32uExpression{})
        case "f64.convert_i32_s":
            return append(subexpressions(expr), &F64ConvertI32sExpression{})
        case "f64.sub":
            return append(subexpressions(expr), &F64SubExpression{})
        case "f64.mul":
            return append(subexpressions(expr), &F64MulExpression{})
        case "f64.div":
            return append(subexpressions(expr), &F64DivExpression{})
        case "f64.copysign":
            return append(subexpressions(expr), &F64CopySignExpression{})
        case "f64.eq":
            return append(subexpressions(expr), &F64EqExpression{})
        case "f64.lt":
            return append(subexpressions(expr), &F64LtExpression{})
        case "f64.gt":
            return append(subexpressions(expr), &F64GtExpression{})
        case "f64.ge":
            return append(subexpressions(expr), &F64GeExpression{})
        case "f64.min":
            return append(subexpressions(expr), &F64MinExpression{})
        case "f64.max":
            return append(subexpressions(expr), &F64MaxExpression{})
        case "f32.mul":
            return append(subexpressions(expr), &F32MulExpression{})
        case "f32.copysign":
            return append(subexpressions(expr), &F32CopySignExpression{})
        case "f32.le":
            return append(subexpressions(expr), &F32LeExpression{})
        case "f32.ge":
            return append(subexpressions(expr), &F32GeExpression{})
        case "f32.min":
            return append(subexpressions(expr), &F32MinExpression{})
        case "f32.max":
            return append(subexpressions(expr), &F32MaxExpression{})
        case "f32.store":
            return append(subexpressions(expr), &F32StoreExpression{})
        case "f32.sqrt":
            return append(subexpressions(expr), &F32SqrtExpression{})
        case "f32.eq":
            return append(subexpressions(expr), &F32EqExpression{})
        case "f32.lt":
            return append(subexpressions(expr), &F32LtExpression{})
        case "f32.add":
            return append(subexpressions(expr), &F32AddExpression{})
        case "f32.sub":
            return append(subexpressions(expr), &F32SubExpression{})
        case "f32.div":
            return append(subexpressions(expr), &F32DivExpression{})
        case "f32.const":
            value, err := parseFloat32(expr.Children[0].Value)
            if err != nil {
                fmt.Printf("Unable to parse float '%v': %v\n", expr.Children[0].Value, err)
                return nil
            }

            return []Expression{
                &F32ConstExpression{
                    N: value,
                },
            }
        case "f64.le":
            return append(subexpressions(expr), &F64LeExpression{})
        case "f64.ne":
            return append(subexpressions(expr), &F64NeExpression{})
        case "f32.ne":
            return append(subexpressions(expr), &F32NeExpression{})
        case "f64.add":
            return append(subexpressions(expr), &F64AddExpression{})
        case "f64.const":
            value, err := parseFloat64(expr.Children[0].Value)
            if err != nil {
                fmt.Printf("Unable to parse float '%v': %v\n", expr.Children[0].Value, err)
            }
            return []Expression{
                &F64ConstExpression{
                    N: value,
                },
            }
        case "f32.neg":
            if len(expr.Children) > 0 {
                return append(MakeExpressions(module, code, labels, expr.Children[0]), &F32NegExpression{})
            }

            return []Expression{&F32NegExpression{}}
        case "f64.neg":
            if len(expr.Children) > 0 {
                return append(MakeExpressions(module, code, labels, expr.Children[0]), &F64NegExpression{})
            }

            return []Expression{&F64NegExpression{}}
        case "local.get":
            name := expr.Children[0].Value
            index, err := strconv.Atoi(name)
            if err != nil {
                var ok bool
                index, ok = code.LookupLocal(name)
                if !ok {
                    fmt.Printf("Error: unable to find named local '%v'\n", name)
                    return nil
                }
            }

            return []Expression{&LocalGetExpression{Local: uint32(index)}}
        case "local.set":
            name := expr.Children[0].Value
            index, err := strconv.Atoi(name)
            if err != nil {
                var ok bool
                index, ok = code.LookupLocal(name)
                if !ok {
                    fmt.Printf("Error: unable to find named local '%v'\n", name)
                    return nil
                }
            }

            var out []Expression
            if len(expr.Children) > 1 {
                out = MakeExpressions(module, code, labels, expr.Children[1])
            }

            return append(out, &LocalSetExpression{Local: uint32(index)})
        case "local.tee":
            name := expr.Children[0].Value
            index, err := strconv.Atoi(expr.Children[0].Value)
            if err != nil {
                var ok bool
                index, ok = code.LookupLocal(name)
                if !ok {
                    fmt.Printf("Error: unable to find named local '%v'\n", name)
                    return nil
                }
            }

            var out []Expression
            if len(expr.Children) > 1 {
                out = MakeExpressions(module, code, labels, expr.Children[1])
            }

            return append(out, &LocalTeeExpression{Local: uint32(index)})
        case "global.get":
            name := expr.Children[0]

            var index uint32
            v, err := strconv.Atoi(name.Value)
            if err != nil {
                var ok bool
                index, ok = module.GetGlobalSection().LookupGlobal(name.Value)
                if !ok {
                    fmt.Printf("Error: unable to find global '%v'\n", name.Value)
                    return nil
                }
            } else {
                index = uint32(v)
            }

            var out []Expression
            if len(expr.Children) > 1 {
                out = MakeExpressions(module, code, labels, expr.Children[1])
            }

            return append(out, &GlobalGetExpression{&GlobalIndex{Id: index}})

        case "global.set":
            name := expr.Children[0]

            var index uint32
            v, err := strconv.Atoi(name.Value)
            if err != nil {
                var ok bool
                index, ok = module.GetGlobalSection().LookupGlobal(name.Value)
                if !ok {
                    fmt.Printf("Error: unable to find global '%v'\n", name.Value)
                    return nil
                }
            } else {
                index = uint32(v)
            }

            var out []Expression
            if len(expr.Children) > 1 {
                out = MakeExpressions(module, code, labels, expr.Children[1])
            }

            return append(out, &GlobalSetExpression{&GlobalIndex{Id: index}})
        case "f32.gt":
            return append(subexpressions(expr), &F32GtExpression{})
        case "f32.load":
            return append(subexpressions(expr), &F32LoadExpression{})
        case "i32.load":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            // FIXME: handle memory argument alignment and offset
            return append(out, &I32LoadExpression{MemoryArgument{}})
        case "i32.load8_s":
            // FIXME: handle memory argument alignment and offset
            return append(subexpressions(expr), &I32Load8sExpression{MemoryArgument{}})
        case "i32.load8_u":
            return append(subexpressions(expr), &I32Load8uExpression{})
        case "i64.div_s":
            return append(subexpressions(expr), &I64DivsExpression{})
        case "i64.div_u":
            return append(subexpressions(expr), &I64DivuExpression{})
        case "i64.rem_s":
            return append(subexpressions(expr), &I64RemsExpression{})
        case "i64.rem_u":
            return append(subexpressions(expr), &I64RemuExpression{})
        case "i64.and":
            return append(subexpressions(expr), &I64AndExpression{})
        case "i64.or":
            return append(subexpressions(expr), &I64OrExpression{})
        case "i64.xor":
            return append(subexpressions(expr), &I64XOrExpression{})
        case "i64.shl":
            return append(subexpressions(expr), &I64ShlExpression{})
        case "i64.shr_u":
            return append(subexpressions(expr), &I64ShruExpression{})
        case "i64.shr_s":
            return append(subexpressions(expr), &I64ShrsExpression{})
        case "i64.ne":
            return append(subexpressions(expr), &I64NeExpression{})
        case "i64.le_s":
            return append(subexpressions(expr), &I64LesExpression{})
        case "i64.ge_s":
            return append(subexpressions(expr), &I64GesExpression{})
        case "i64.ge_u":
            return append(subexpressions(expr), &I64GeuExpression{})
        case "i64.store8":
            return append(subexpressions(expr), &I64Store8Expression{})
        case "i64.store32":
            return append(subexpressions(expr), &I64Store8Expression{})
        case "i64.load8_s":
            return append(subexpressions(expr), &I64Load8sExpression{MemoryArgument{}})
        case "f64.store":
            return append(subexpressions(expr), &F64StoreExpression{})
        case "i64.store":
            return append(subexpressions(expr), &I64StoreExpression{})
        case "i64.store16":
            return append(subexpressions(expr), &I64Store16Expression{})
        case "i32.store":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            // FIXME: handle memory argument alignment and offset
            return append(out, &I32StoreExpression{})
        case "i32.store8":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            // FIXME: handle memory argument alignment and offset
            return append(out, &I32Store8Expression{})
        case "i32.store16":
            var out []Expression
            for _, child := range expr.Children {
                out = append(out, MakeExpressions(module, code, labels, child)...)
            }

            // FIXME: handle memory argument alignment and offset
            return append(out, &I32Store16Expression{})

    }

    fmt.Printf("Warning: unhandled wast expression '%v'\n", expr)

    return nil
}

/* FIXME: remove */
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
            expressions := MakeExpressions(module, &out, data.Stack[string]{}, current)
            out.Expressions = append(out.Expressions, expressions...)
        }
    }

    return out
}

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

func doSecondPassExpression(expr Expression) Expression {
    switch expr.(type) {
        case *SecondPassExpression:
            second := expr.(*SecondPassExpression)
            return second.Replace()
        case *BlockExpression:
            block := expr.(*BlockExpression)

            for i := 0; i < len(block.Instructions); i++ {
                block.Instructions[i] = doSecondPassExpression(block.Instructions[i])
            }

            for i := 0; i < len(block.ElseInstructions); i++ {
                block.ElseInstructions[i] = doSecondPassExpression(block.ElseInstructions[i])
            }

            return block
        default:
            return expr
    }
}

func doSecondPass(code *Code){
    for i := 0; i < len(code.Expressions); i++ {
        code.Expressions[i] = doSecondPassExpression(code.Expressions[i])
    }
}

func CreateWasmModule(module *sexp.SExpression) (WebAssemblyModule, error) {
    var moduleOut WebAssemblyModule
    typeSection := NewWebAssemblyTypeSection()
    functionSection := WebAssemblyFunctionSectionCreate()
    codeSection := new(WebAssemblyCodeSection)
    tableSection := new(WebAssemblyTableSection)
    exportSection := new(WebAssemblyExportSection)
    memorySection := new(WebAssemblyMemorySection)
    globalSection := new(WebAssemblyGlobalSection)
    elementSection := new(WebAssemblyElementSection)

    moduleOut.AddSection(typeSection)
    moduleOut.AddSection(functionSection)
    moduleOut.AddSection(codeSection)
    moduleOut.AddSection(tableSection)
    moduleOut.AddSection(elementSection)
    moduleOut.AddSection(globalSection)
    moduleOut.AddSection(memorySection)
    moduleOut.AddSection(exportSection)

    for _, expr := range module.Children {
        /*
        if expr.Name == "func" {
            fmt.Printf("Func: %v\n", expr)
        }
        */

        switch expr.Name {
            case "func":
                var code Code
                var functionType WebAssemblyFunction
                var functionName string
                var exportedName string

                for i, child := range expr.Children {
                    /* named function */
                    if i == 0 && child.Value != "" {
                        functionName = child.Value
                    } else {
                        switch child.Name {
                            case "type":
                                if len(child.Children) > 0 {
                                    if child.Children[0].Value != "" {
                                        type_ := typeSection.GetTypeByName(child.Children[0].Value)
                                        if type_ != nil {
                                            functionType = typeSection.GetFunction(type_.Id)

                                            for _, parameter := range functionType.InputTypes {
                                                code.Locals = append(code.Locals, Local{
                                                    Count: 1,
                                                    Name: parameter.Name,
                                                    Type: parameter.Type,
                                                })
                                            }
                                        }
                                    }
                                }
                            case "export":
                                exportedName = cleanName(child.Children[0].Value)
                            case "param":
                                var paramName string
                                for i, param := range child.Children {
                                    use := ValueTypeFromName(param.Value)
                                    if i == 0 && use == InvalidValueType {
                                        paramName = param.Value
                                    } else {
                                        code.Locals = append(code.Locals, Local{
                                            Count: 1,
                                            Name: paramName,
                                            Type: use,
                                        })

                                        functionType.InputTypes = append(functionType.InputTypes, Parameter{
                                            Name: paramName,
                                            Type: use,
                                        })

                                        paramName = ""
                                    }
                                }
                                // functionType.InputTypes = append(functionType.InputTypes, ConvertValueTypes(child)...)
                            case "result":
                                functionType.OutputTypes = ConvertValueTypes(child)
                            case "local":
                                if len(child.Children) > 0 {
                                    var firstLocalName string
                                    start := 0
                                    if ValueTypeFromName(child.Children[0].Value) == InvalidValueType {
                                        firstLocalName = child.Children[0].Value
                                        start = 1
                                    }
                                    for i := start; i < len(child.Children); i++ {
                                        name := ""
                                        if i == start {
                                            name = firstLocalName
                                        }
                                        code.Locals = append(code.Locals, Local{
                                            Count: 1,
                                            Name: name,
                                            Type: ValueTypeFromName(child.Children[i].Value),
                                        })
                                    }
                                }
                            default:
                                code.Expressions = append(code.Expressions, MakeExpressions(moduleOut, &code, data.Stack[string]{}, child)...)
                        }
                    }
                }

                typeIndex := typeSection.GetOrCreateFunctionType(functionType)
                functionIndex := functionSection.AddFunction(&TypeIndex{
                    Id: typeIndex,
                }, cleanName(functionName))

                codeSection.AddCode(code)
                if exportedName != "" {
                    exportSection.AddExport(exportedName, &FunctionIndex{Id: functionIndex})
                }
            case "type":
                name := expr.Children[0]
                kind := expr.Children[1]
                if kind.Name == "func" {
                    typeIndex := typeSection.GetOrCreateFunctionType(MakeFunctionType(kind))
                    typeSection.AssociateName(name.Value, &TypeIndex{Id: typeIndex})
                }
            case "global":
                name := expr.Children[0]
                kind := expr.Children[1]
                value := expr.Children[2]

                globalType := GlobalType{}

                if kind.Name == "mut" {
                    globalType.Mutable = true
                    globalType.ValueType = ValueTypeFromName(kind.Children[0].Value)
                } else {
                    globalType.Mutable = false
                    globalType.ValueType = ValueTypeFromName(kind.Value)
                }

                valueExpr := MakeExpressions(moduleOut, nil, data.Stack[string]{}, value)
                globalSection.AddGlobal(&globalType, valueExpr, name.Value)
            case "memory":
                min, err := strconv.Atoi(expr.Children[0].Value)
                if err != nil {
                    fmt.Printf("Error: unable to read minimum length of memory: %v", err)
                    break
                }

                memorySection.AddMemory(Limit{Minimum: uint32(min)})

            case "table":
                // so far this handles an inline table expression with funcref elements already given
                tableName := ""
                for i := 0; i < len(expr.Children); i++ {
                    if expr.Children[i].Value == "funcref" {
                        if len(expr.Children) > i+1 {
                            elements := expr.Children[i+1]
                            if elements.Name == "elem" {
                                tableId := tableSection.AddTable(TableType{
                                    Limit: Limit{
                                        Minimum: uint32(len(elements.Children)),
                                        Maximum: uint32(len(elements.Children)),
                                        HasMaximum: true,
                                    },
                                    RefType: RefTypeFunction,
                                    Name: tableName,
                                })

                                /* initialize the elements after the module has been parsed */
                                defer func(){
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
                                }()
                            }
                        }

                        break
                    } else {
                        tableName = expr.Children[i].Value
                    }
                }

            default:
                fmt.Printf("Warning: unhandled wast top level '%v'\n", expr.Name)
        }
    }

    for _, code := range codeSection.Code {
        doSecondPass(&code)
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
        
        wast.Expressions = append(wast.Expressions, next)
    }

    return wast, nil
}
