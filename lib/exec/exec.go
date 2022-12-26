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

func (value RuntimeValue) String() string {
    switch value.Kind {
        case RuntimeValueNone: return "none"
        case RuntimeValueI32: return fmt.Sprintf("%v:i32", value.I32)
        case RuntimeValueI64: return fmt.Sprintf("%v:i64", value.I64)
        case RuntimeValueF32: return fmt.Sprintf("%v:f32", value.F32)
        case RuntimeValueF64: return fmt.Sprintf("%v:f64", value.F64)
    }

    return "?"
}

func Trap(reason string) error {
    return fmt.Errorf(reason)
}

func Execute(stack *data.Stack[RuntimeValue], labels *data.Stack[int], expressions []core.Expression, instruction int) (int, int, error) {
    current := expressions[instruction]

    fmt.Printf("Stack is now %+v\n", *stack)

    /* branches work by jumping to the Nth block, where N represents the nesting level of the blocks.
     *   (block_a (block_b (br 0) (br 1)))
     * Here the blocks are annotated with 'a' and 'b' to make their references more clear, but in actual
     * syntax the _a and _b would be left off.
     *
     * The (br 0) would jump to the end of block_b, and the (br 1) would jump to the end of block_a. The number
     * after the br represents how many nested blocks to jump back through.
     *
     * This interpreter implements branches as the second return value in (a,b,c) returning N+1 where N is the 0 in (br 0).
     * When a block receives a non-zero value for b after executing an instruction, the block will immediately return
     * with b-1, thus allowing the previous block to also abort early.
     */
    switch current.(type) {
        case *core.BlockExpression:
            block := current.(*core.BlockExpression)

            /* Keep track of the number of values on the stack in case they need to be popped off later */
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

                    /* Remove all values on the stack that were produced during the dynamic extent of this block
                     */
                    size := labels.Pop()
                    stack.Reduce(size)
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
