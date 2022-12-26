package exec

import (
    "fmt"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/data"
)

type RuntimeValueKind int

const (
    RuntimeValueNone = 0
    RuntimeValueI32 = 1
    RuntimeValueI64 = 2
    RuntimeValueF32 = 3
    RuntimeValueF64 = 4
)

type RuntimeValue struct {
    Kind RuntimeValueKind
    I32 int32
    I64 int64
    F32 float32
    F64 float64
}

func Trap(reason string) error {
    return fmt.Errorf(reason)
}

func Execute(stack *data.Stack[RuntimeValue], labels *data.Stack[int], expressions []core.Expression, instruction int) (int, int, error) {
    current := expressions[instruction]

    fmt.Printf("Stack is now %+v\n", *stack)

    switch current.(type) {
        case *core.BlockExpression:
            block := current.(*core.BlockExpression)

            labels.Push(stack.Size())
            local := 0
            for local < len(block.Instructions) {
                var branch int
                var err error
                local, branch, err = Execute(stack, labels, block.Instructions, local)
                if err != nil {
                    return 0, 0, err
                }
                if branch > 0 {
                    fmt.Printf("Branch to %v\n", branch)
                    last := stack.Pop()

                    size := labels.Pop()
                    for stack.Size() > size {
                        stack.Pop()
                    }

                    stack.Push(last)

                    return instruction+1, branch-1, nil
                }
            }
            labels.Pop()
        case *core.I32ConstExpression:
            expr := current.(*core.I32ConstExpression)
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI32,
                I32: expr.N,
            })
        case *core.BranchExpression:
            expr := current.(*core.BranchExpression)
            return 0, int(expr.Label)+1, nil
        case *core.BranchIfExpression:
            expr := current.(*core.BranchIfExpression)
            value := stack.Pop()
            if value.Kind == RuntimeValueNone {
                return 0, 0, Trap("no value on stack for br_if")
            }

            if value.Kind != RuntimeValueI32 {
                return 0, 0, Trap(fmt.Sprintf("top of stack was not an i32: %+v", value))
            }

            if value.I32 == 0 {
                // nothing, fall through
            } else {
                return 0, int(expr.Label)+1, nil
            }
    }

    return instruction + 1, 0, nil
}

func EvaluateOne(expression core.Expression) (RuntimeValue, error) {
    var stack data.Stack[RuntimeValue]
    var labels data.Stack[int]

    _, _, err := Execute(&stack, &labels, []core.Expression{expression}, 0)
    if err != nil {
        return RuntimeValue{}, err
    }

    if stack.Size() == 0 {
        return RuntimeValue{}, fmt.Errorf("did not produce any values")
    }

    /* FIXME: handle multiple values on the stack */
    return stack.Pop(), nil
}

func RunCode(code core.Code) (RuntimeValue, error) {
    var stack data.Stack[RuntimeValue]
    var labels data.Stack[int]

    instruction := 0

    for instruction < len(code.Expressions) {
        var branch int
        var err error
        instruction, branch, err = Execute(&stack, &labels, code.Expressions, instruction)
        if err != nil {
            return RuntimeValue{}, err
        }
        if branch != 0 {
            return RuntimeValue{}, fmt.Errorf("Branch to non-existent block: %v", branch)
        }
    }

    if stack.Size() > 0 {
        /* FIXME: return all values on the stack */
        return stack.Pop(), nil
    }

    return RuntimeValue{}, nil
}

func Invoke(module core.WebAssemblyModule, name string) (RuntimeValue, error) {
    kind := module.GetExportSection().FindExportByName(name)
    if kind == nil {
        return RuntimeValue{}, fmt.Errorf("no such exported function '%v'", name)
    }

    function, ok := kind.(*core.FunctionIndex)
    if ok {
        code := module.GetCodeSection().GetFunction(function.Id)
        return RunCode(code)
    } else {
        return RuntimeValue{}, fmt.Errorf("no such exported function '%v'", name)
    }
   
    return RuntimeValue{}, fmt.Errorf("shouldnt get here")
}
