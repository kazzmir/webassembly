package exec

import (
    "fmt"
    "strings"
    "reflect"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/data"
    "github.com/kazzmir/webassembly/lib/sexp"
)

type RuntimeValueKind int

const (
    RuntimeValueNone = 0
    RuntimeValueI32 = 1
    RuntimeValueI64 = 2
    RuntimeValueF32 = 3
    RuntimeValueF64 = 4
)

/* represents values during the execution of the webassembly virtual machine */
type RuntimeValue struct {
    Kind RuntimeValueKind
    I32 int32
    I64 int64
    F32 float32
    F64 float64
}

func MakeRuntimeValue(kind core.ValueType) RuntimeValue {
    switch kind {
        case core.ValueTypeI32: return RuntimeValue{Kind: RuntimeValueI32}
        case core.ValueTypeI64: return RuntimeValue{Kind: RuntimeValueI64}
        case core.ValueTypeF32: return RuntimeValue{Kind: RuntimeValueF32}
        case core.ValueTypeF64: return RuntimeValue{Kind: RuntimeValueF64}
    }

    return RuntimeValue{Kind: RuntimeValueNone}
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

type Table struct {
    Elements []core.Index
}

type Global struct {
    Name string
    Value RuntimeValue
    Mutable bool
}

type Store struct {
    Tables []Table
    Globals []Global
}

func InitializeStore(module core.WebAssemblyModule) *Store {
    var out Store

    tableSection := module.GetTableSection()
    if tableSection != nil {
        for _, table := range tableSection.Items {
            out.Tables = append(out.Tables, Table{Elements: make([]core.Index, table.Limit.Minimum)})
        }
    }

    globalSection := module.GetGlobalSection()
    if globalSection != nil {
        for _, global := range globalSection.Globals {

            value, err := EvaluateOne(global.Expression[0])
            if err != nil {
                fmt.Printf("Error: unable to evaluate global: %v\n", err)
                continue
            }

            out.Globals = append(out.Globals, Global{
                Name: global.Name,
                Value: value,
                Mutable: global.Global.Mutable,
            })
        }
    }

    elementSection := module.GetElementSection()
    if elementSection != nil {
        for _, element := range elementSection.Elements {
            switch element.Mode.(type) {
                case *core.ElementModeActive:
                    active := element.Mode.(*core.ElementModeActive)
                    for i, item := range element.Inits {
                        switch item.(type) {
                            case *core.RefFuncExpression:
                                function := item.(*core.RefFuncExpression)
                                out.Tables[active.Table].Elements[i] = function.Function
                        }
                    }
            }
        }
    }

    return &out
}

/* activation frame: https://webassembly.github.io/spec/core/exec/runtime.html#syntax-frame */
type Frame struct {
    Locals []RuntimeValue
    Module core.WebAssemblyModule
}

func Trap(reason string) error {
    return fmt.Errorf(reason)
}

// magic value meaning we are returning from the function rather than just exiting a block
const ReturnLabel int = 1<<30

/* execute a single instruction
 *  input: stack of runtime values, stack of block labels, list of expressions to execute, instruction index into 'expressions', activation frame
 *  output: next instruction number to execute, number of blocks to skip (if greater than 0), and any errors that may occur (including traps)
 */
func Execute(stack *data.Stack[RuntimeValue], labels *data.Stack[int], expressions []core.Expression, instruction int, frame Frame, store *Store) (int, int, error) {
    current := expressions[instruction]

    // fmt.Printf("Stack is now %+v\n", *stack)

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

            instructions := block.Instructions

            /* for an if block, pop a value off the stack and if its not 0 then execute the normal instructions,
             * otherwise execute the else instructions.
             */
            if block.Kind == core.BlockKindIf {
                value := stack.Pop()
                if value.I32 == 0 {
                    instructions = block.ElseInstructions
                }
            }

            /* Keep track of the number of values on the stack in case they need to be popped off later */
            labels.Push(stack.Size())
            local := 0
            for local < len(instructions) {
                var branch int
                var err error
                local, branch, err = Execute(stack, labels, instructions, local, frame, store)
                if err != nil {
                    return 0, 0, err
                }

                if branch > 0 {
                    // fmt.Printf("Branch to %v\n", branch)
                    last := stack.Pop()

                    /* Remove all values on the stack that were produced during the dynamic extent of this block
                     */
                    size := labels.Pop()
                    stack.Reduce(size)
                    stack.Push(last)

                    /* if we are handling a return then don't change the branch value so that all parent blocks
                     * also do a return
                     */
                    if branch == ReturnLabel {
                        return instruction+1, branch, nil
                    }

                    /* go back to the same block instruction if we are branching to this loop */
                    if branch == 1 && block.Kind == core.BlockKindLoop {
                        return instruction, 0, nil
                    }

                    /* otherwise go to the instruction after this block */
                    return instruction+1, branch-1, nil
                }
            }
            labels.Pop()

        case *core.SelectExpression:
            c := stack.Pop()

            v1 := stack.Pop()
            v2 := stack.Pop()
            if c.I32 != 0 {
                stack.Push(v1)
            } else {
                stack.Push(v2)
            }
        case *core.I32ConstExpression:
            expr := current.(*core.I32ConstExpression)
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI32,
                I32: expr.N,
            })
        case *core.I64ConstExpression:
            expr := current.(*core.I64ConstExpression)
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI64,
                I64: expr.N,
            })
        case *core.F32ConstExpression:
            expr := current.(*core.F32ConstExpression)
            stack.Push(RuntimeValue{
                Kind: RuntimeValueF32,
                F32: expr.N,
            })
        case *core.F64ConstExpression:
            expr := current.(*core.F64ConstExpression)
            stack.Push(RuntimeValue{
                Kind: RuntimeValueF64,
                F64: expr.N,
            })
        case *core.I32AddExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI32,
                I32: arg1.I32 + arg2.I32,
            })
        case *core.DropExpression:
            stack.Pop()
        case *core.ReturnExpression:
            return 0, ReturnLabel, nil
        case *core.BranchTableExpression:
            expr := current.(*core.BranchTableExpression)
            value := stack.Pop()

            if value.Kind != RuntimeValueI32 {
                return 0, 0, Trap(fmt.Sprintf("top of stack was not an i32: %+v", value))
            }

            /* FIXME: what to do here? */
            if len(expr.Labels) == 0 {
                return 0, 0, fmt.Errorf("br_table had no labels")
            }

            if int(value.I32) < len(expr.Labels) {
                return 0, int(expr.Labels[value.I32])+1, nil
            }

            return 0, int(expr.Labels[len(expr.Labels)-1])+1, nil

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
        case *core.LocalGetExpression:
            expr := current.(*core.LocalGetExpression)

            if len(frame.Locals) <= int(expr.Local) {
                return 0, 0, fmt.Errorf("unable to get local %v when frame has %v locals", expr.Local, len(frame.Locals))
            }
            stack.Push(frame.Locals[expr.Local])
        case *core.LocalTeeExpression:
            expr := current.(*core.LocalTeeExpression)

            if len(frame.Locals) <= int(expr.Local) {
                return 0, 0, fmt.Errorf("unable to tee local %v when frame has %v locals", expr.Local, len(frame.Locals))
            }
            value := stack.Pop()
            stack.Push(frame.Locals[expr.Local])
            stack.Push(frame.Locals[expr.Local])
            frame.Locals[expr.Local] = value

        case *core.LocalSetExpression:
            expr := current.(*core.LocalSetExpression)

            if len(frame.Locals) <= int(expr.Local) {
                return 0, 0, fmt.Errorf("unable to set local %v when frame has %v locals", expr.Local, len(frame.Locals))
            }
            value := stack.Pop()
            frame.Locals[expr.Local] = value

        case *core.GlobalSetExpression:
            expr := current.(*core.GlobalSetExpression)
            index := expr.Global.Id

            if int(index) >= len(store.Globals) {
                return 0, 0, fmt.Errorf("unable to set global %v when store has %v globals", index, len(store.Globals))
            }

            if !store.Globals[index].Mutable {
                return 0, 0, fmt.Errorf("global %v is not mutable", index)
            }

            value := stack.Pop()
            store.Globals[index].Value = value

        case *core.CallExpression:
            /* create a new stack frame, pop N values off the stack and put them in the locals of the frame.
             * then invoke the code of the function with the new frame.
             * put the resulting runtime value back on the stack
             */
            expr := current.(*core.CallExpression)

            functionTypeIndex := frame.Module.GetFunctionSection().GetFunctionType(int(expr.Index.Id))
            functionType := frame.Module.GetTypeSection().GetFunction(functionTypeIndex.Id)

            var args []RuntimeValue
            for _, input := range functionType.InputTypes {
                // FIXME: check that the stack contains the same type as 'input'
                _ = input
                args = append(args, stack.Pop())
            }

            code := frame.Module.GetCodeSection().GetFunction(expr.Index.Id)

            for _, local := range code.Locals {
                args = append(args, MakeRuntimeValue(local.Type))
            }

            out, err := RunCode(code, Frame{
                Locals: args,
                Module: frame.Module,
            }, store)

            if err != nil {
                return 0, 0, err
            }

            stack.Push(out)

        default:
            return 0, 0, fmt.Errorf("unhandled instruction %v %+v", reflect.TypeOf(current), current)
    }

    return instruction + 1, 0, nil
}

/* evaluate a single expression and return whatever runtimevalue the expression produces */
func EvaluateOne(expression core.Expression) (RuntimeValue, error) {
    var stack data.Stack[RuntimeValue]
    var labels data.Stack[int]

    _, _, err := Execute(&stack, &labels, []core.Expression{expression}, 0, Frame{}, nil)
    if err != nil {
        return RuntimeValue{}, err
    }

    if stack.Size() == 0 {
        return RuntimeValue{}, fmt.Errorf("did not produce any values")
    }

    /* FIXME: handle multiple values on the stack */
    return stack.Pop(), nil
}

/* evaluate an entire function
 */
func RunCode(code core.Code, frame Frame, store *Store) (RuntimeValue, error) {
    var stack data.Stack[RuntimeValue]
    var labels data.Stack[int]

    instruction := 0

    for instruction < len(code.Expressions) {
        var branch int
        var err error
        instruction, branch, err = Execute(&stack, &labels, code.Expressions, instruction, frame, store)
        if err != nil {
            return RuntimeValue{}, err
        }
        if branch == ReturnLabel {
            return stack.Pop(), nil
        }
        /* if we a branch was executed to label 0, then the 'branch' variable will be equal to 1,
         * which has the same meaning as the end of the function
         */
        if branch == 1 {
            return stack.Pop(), nil
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

/* invoke an exported function in the given module */
func Invoke(module core.WebAssemblyModule, store *Store, name string, args []RuntimeValue) (RuntimeValue, error) {
    kind := module.GetExportSection().FindExportByName(name)
    if kind == nil {
        return RuntimeValue{}, fmt.Errorf("no such exported function '%v'", name)
    }

    function, ok := kind.(*core.FunctionIndex)
    if ok {
        code := module.GetCodeSection().GetFunction(function.Id)

        for _, local := range code.Locals {
            args = append(args, MakeRuntimeValue(local.Type))
        }

        frame := Frame{
            Locals: args,
            Module: module,
        }

        return RunCode(code, frame, store)
    } else {
        return RuntimeValue{}, fmt.Errorf("no such exported function '%v'", name)
    }
   
    return RuntimeValue{}, fmt.Errorf("shouldnt get here")
}

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

/* handle wast-style (assert_return ...) */
func AssertReturn(module core.WebAssemblyModule, assert sexp.SExpression) error {
    what := assert.Children[0]
    if what.Name == "invoke" {

        functionName := cleanName(what.Children[0].Value)

        var args []RuntimeValue
        for _, arg := range what.Children[1:] {
            expressions := core.MakeExpressions(module, arg)
            nextArg, err := EvaluateOne(expressions[0])
            if err != nil {
                return err
            }

            args = append(args, nextArg)
        }

        store := InitializeStore(module)

        result, err := Invoke(module, store, functionName, args)
        if err != nil {
            return err
        }

        if len(assert.Children) == 2 {
            expressions := core.MakeExpressions(module, assert.Children[1])
            expected, err := EvaluateOne(expressions[0])
            if err != nil {
                return err
            } else {
                if result != expected {
                    return fmt.Errorf("result=%v expected=%v", result, expected)
                }
            }
        }
    }

    return nil
}
