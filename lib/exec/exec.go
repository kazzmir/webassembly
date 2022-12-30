package exec

import (
    "fmt"
    "strings"
    "reflect"
    "math"
    "math/bits"
    "encoding/binary"
    "github.com/kazzmir/webassembly/lib/core"
    "github.com/kazzmir/webassembly/lib/data"
    "github.com/kazzmir/webassembly/lib/sexp"
)

const MemoryPageSize = 65536

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
    Memory [][]byte
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

    memorySection := module.GetMemorySection()
    if memorySection != nil {
        for _, memory := range memorySection.Memories {
            data := make([]byte, memory.Minimum * MemoryPageSize)
            out.Memory = append(out.Memory, data)
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

func i32(value int32) RuntimeValue {
    return RuntimeValue{
        Kind: RuntimeValueI32,
        I32: value,
    }
}

func i64(value int64) RuntimeValue {
    return RuntimeValue{
        Kind: RuntimeValueI64,
        I64: value,
    }
}

func f32(value float32) RuntimeValue {
    return RuntimeValue{
        Kind: RuntimeValueF32,
        F32: value,
    }
}

func f64(value float64) RuntimeValue {
    return RuntimeValue{
        Kind: RuntimeValueF64,
        F64: value,
    }
}

type ByteWriter struct {
    data []byte
}

func NewByteWriter(data []byte) *ByteWriter {
    return &ByteWriter{data: data}
}

func (writer *ByteWriter) Write(data []byte) (int, error) {
    if len(data) < len(writer.data) {
        return copy(writer.data, data), nil
    } else {
        return 0, fmt.Errorf("buffer too small to write to")
    }
}

// magic value meaning we are returning from the function rather than just exiting a block
const ReturnLabel int = 1<<30

/* execute a single instruction
 *  input: stack of runtime values, stack of block labels, list of expressions to execute, instruction index into 'expressions', activation frame
 *  output: next instruction number to execute, number of blocks to skip (if greater than 0), and any errors that may occur (including traps)
 */
// FIXME: remove labels
func Execute(stack *data.Stack[RuntimeValue], labels *data.Stack[int], expressions []core.Expression, instruction int, frame Frame, store *Store) (int, int, error) {
    current := expressions[instruction]

    /*
    fmt.Printf("Stack is now %+v executing %v\n", *stack, reflect.TypeOf(current))
    if stack.Size() > 100 {
        return 0, 0, fmt.Errorf("fail")
    }
    */

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
                if value.Kind != RuntimeValueI32 {
                    return 0, 0, fmt.Errorf("if expected an i32 on the stack but got %v", value)
                }
                if value.I32 == 0 {
                    instructions = block.ElseInstructions
                }
            }

            currentStackSize := stack.Size()

            /* Keep track of the number of values on the stack in case they need to be popped off later */
            // labels.Push(stack.Size())
            local := 0
            for local < len(instructions) {
                var branch int
                var err error
                local, branch, err = Execute(stack, labels, instructions, local, frame, store)
                if err != nil {
                    return 0, 0, err
                }

                if branch > 0 {

                    /* if we are handling a return then don't change the branch value so that all parent blocks
                     * also do a return. the stack will contain all kinds of stuff, but the function return
                     * will pop off the values it needs
                     */
                    if branch == ReturnLabel {
                        // labels.Pop()
                        return instruction+1, branch, nil
                    }

                    /* go back to the same block instruction if we are branching to this loop */
                    if branch == 1 && block.Kind == core.BlockKindLoop {
                        /* don't pop anything from the stack if we are re-entering the loop */
                        // labels.Pop()
                        return instruction, 0, nil
                    }

                    // fmt.Printf("Branch to %v\n", branch)
                    if branch == 1 {
                        /* we are jumping back to some block that only cares about the last N values on the stack, so pop all values
                         * between whatever was on the stack when the block was entered and what is there now
                         */
                        results := stack.PopN(len(block.ExpectedType))

                        for _, last := range results {
                            if last.Kind == RuntimeValueNone {
                                return 0, 0, fmt.Errorf("invalid runtime value on stack after branch")
                            }
                        }

                        /* Remove all values on the stack that were produced during the dynamic extent of this block
                         */
                        // size := labels.Pop()
                        size := currentStackSize
                        stack.Reduce(size)
                        stack.PushAll(results)
                    } else {
                        // labels.Pop()
                    }

                    /* otherwise go to the instruction after this block */
                    return instruction+1, branch-1, nil
                }
            }
            // labels.Pop()

        case *core.UnreachableExpression:
            return 0, 0, Trap("unreachable")
        case *core.SelectExpression:
            c := stack.Pop()

            v2 := stack.Pop()
            v1 := stack.Pop()
            if c.I32 != 0 {
                stack.Push(v1)
            } else {
                stack.Push(v2)
            }
        case *core.I32MulExpression:
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI32,
                I32: stack.Pop().I32 * stack.Pop().I32,
            })
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
        case *core.I64LeuExpression:
            a := stack.Pop()
            b := stack.Pop()
            if uint64(b.I64) <= uint64(a.I64) {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32LeuExpression:
            a := stack.Pop()
            b := stack.Pop()
            if uint32(b.I32) <= uint32(a.I32) {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32LesExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.I32 <= a.I32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32NeExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.I32 != a.I32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32Extend8sExpression:
            stack.Push(i32(int32(int8(stack.Pop().I32))))
        case *core.I32Extend16sExpression:
            stack.Push(i32(int32(int16(stack.Pop().I32))))
        case *core.I64ExtendI32sExpression:
            value := stack.Pop()
            stack.Push(i64(int64(value.I32)))
        case *core.F64AddExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(f64(a.F64 + b.F64))
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
        case *core.F64LeExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F64 < a.F64 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F32EqExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F32 == a.F32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F64NeExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F64 != a.F64 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F32NeExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F32 != a.F32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F32DivExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(f32(b.F32 / a.F32))
        case *core.F32SubExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(f32(b.F32 - a.F32))
        case *core.F32AddExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(f32(b.F32 + a.F32))
        case *core.F32SqrtExpression:
            value := stack.Pop()
            stack.Push(f32(float32(math.Sqrt(float64(value.F32)))))
        case *core.F32GtExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F32 > a.F32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F32LtExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.F32 < a.F32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.F32NegExpression:
            stack.Push(f32(-stack.Pop().F32))
        case *core.F64NegExpression:
            stack.Push(f64(-stack.Pop().F64))
        case *core.F64ConvertI64uExpression:
            stack.Push(f64(float64(stack.Pop().I64)))
        case *core.F64ConvertI32uExpression:
            /* FIXME: check this */
            stack.Push(f64(float64(uint32(stack.Pop().I32))))
        case *core.F64ConvertI32sExpression:
            stack.Push(f64(float64(stack.Pop().I32)))
        case *core.I64TruncF64sExpression:
            /* FIXME: handle NaN and infinity */
            stack.Push(i64(int64(stack.Pop().F64)))
        case *core.F64PromoteF32Expression:
            /* FIXME: handle NaN stuff */
            stack.Push(f64(float64(stack.Pop().F32)))
        case *core.I32WrapI64Expression:
            /* FIXME: not sure about this one */
            value := stack.Pop()
            stack.Push(i32(int32(value.I64 % int64(math.MaxInt32))))
        case *core.I32LtuExpression:
            a := stack.Pop()
            b := stack.Pop()
            if uint32(b.I32) < uint32(a.I32) {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32LtsExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.I32 < a.I32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.I32EqExpression:
            a := stack.Pop()
            b := stack.Pop()
            if b.I32 == a.I32 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
        case *core.MemoryGrowExpression:
            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory defined for grow")
            }

            size := stack.Pop()

            limit := frame.Module.GetMemorySection().Memories[0]
            if limit.HasMaximum && len(store.Memory) / MemoryPageSize + int(size.I32) >= int(limit.Maximum) {
                stack.Push(RuntimeValue{
                    Kind: RuntimeValueI32,
                    I32: -1,
                })
            } else {
                oldSize := len(store.Memory[0]) / MemoryPageSize
                more := make([]byte, size.I32 * MemoryPageSize)
                store.Memory[0] = append(store.Memory[0], more...)
                stack.Push(RuntimeValue{
                    Kind: RuntimeValueI32,
                    I32: int32(oldSize),
                })
            }

        case *core.F32LoadExpression:
            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for f32.load")
            }

            memory := store.Memory[0]

            index := stack.Pop()

            if int(index.I32) >= len(memory) {
                return 0, 0, Trap(fmt.Sprintf("invalid memory index %v", index.I32))
            }

            stack.Push(f32(float32(binary.LittleEndian.Uint32(memory[index.I32:]))))

        case *core.I32LoadExpression:
            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i32.load")
            }

            memory := store.Memory[0]

            index := stack.Pop()

            if int(index.I32) >= len(memory) {
                return 0, 0, Trap(fmt.Sprintf("invalid memory index %v", index.I32))
            }

            stack.Push(i32(int32(binary.LittleEndian.Uint32(memory[index.I32:]))))

        case *core.I32Load8sExpression:
            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i64.load8_s")
            }

            memory := store.Memory[0]

            index := stack.Pop()

            if int(index.I32) >= len(memory) {
                return 0, 0, Trap(fmt.Sprintf("invalid memory index %v", index.I32))
            }

            stack.Push(i32(int32(memory[index.I32])))
        case *core.I64Load8sExpression:
            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i64.load8_s")
            }

            memory := store.Memory[0]

            index := stack.Pop()

            if int(index.I32) >= len(memory) {
                return 0, 0, Trap(fmt.Sprintf("invalid memory index %v", index.I32))
            }

            stack.Push(i64(int64(memory[index.I32])))
        case *core.I32StoreExpression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i32.store")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            binary.LittleEndian.PutUint32(memory[index.I32:], uint32(value.I32))
        case *core.I32Store16Expression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i32.store")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            binary.LittleEndian.PutUint16(memory[index.I32:], uint16(int16(value.I32)))
        case *core.I32Store8Expression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i32.store")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            memory[index.I32] = byte(value.I32)
        case *core.I64StoreExpression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i64.store")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            binary.LittleEndian.PutUint64(memory[index.I32:], uint64(value.I64))
        case *core.I64Store16Expression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i64.store16")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            binary.LittleEndian.PutUint16(memory[index.I32:], uint16(int16(value.I64)))
        case *core.F64StoreExpression:
            value := stack.Pop()
            index := stack.Pop()

            if len(store.Memory) == 0 {
                return 0, 0, fmt.Errorf("no memory available for i32.store")
            }

            memory := store.Memory[0]

            if int(index.I32) >= len(memory) {
                return 0, 0, fmt.Errorf("invalid memory location: %v", index)
            }

            binary.Write(NewByteWriter(memory[index.I32:]), binary.LittleEndian, value.F64)
        case *core.I32EqzExpression:
            value := stack.Pop()
            result := 0
            if value.I32 == 0 {
                result = 1
            }

            stack.Push(i32(int32(result)))
        case *core.I64EqzExpression:
            value := stack.Pop()
            result := 0
            if value.I64 == 0 {
                result = 1
            }

            stack.Push(i32(int32(result)))

        case *core.I32DivsExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(i32(b.I32 / a.I32))
        case *core.I32DivuExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(i32(int32(uint32(b.I32) / uint32(a.I32))))
        case *core.I32CtzExpression:
            value := stack.Pop()
            stack.Push(i32(int32(bits.TrailingZeros32(uint32(value.I32)))))
        case *core.I32ClzExpression:
            value := stack.Pop()
            stack.Push(i32(int32(bits.LeadingZeros32(uint32(value.I32)))))
        case *core.I32PopcntExpression:
            stack.Push(i32(int32(bits.OnesCount32(uint32(stack.Pop().I32)))))
        case *core.I32RemsExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(i32(b.I32 % a.I32))
        case *core.I32RemuExpression:
            a := stack.Pop()
            b := stack.Pop()
            stack.Push(i32(int32(uint32(b.I32) % uint32(a.I32))))
        case *core.I32AndExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg1.I32 & arg2.I32))
        case *core.I32OrExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg1.I32 | arg2.I32))
        case *core.I32XOrExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg1.I32 ^ arg2.I32))
        case *core.I32ShlExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg2.I32 << uint32(arg1.I32)))
        case *core.I32ShlsExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg2.I32 << arg1.I32))
        case *core.I32ShluExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg2.I32 << uint32(arg1.I32)))
        case *core.I32ShrsExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg2.I32 >> uint32(arg1.I32)))
        case *core.I32ShruExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg2.I32 >> uint32(arg1.I32)))
        case *core.I32RotlExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(int32(bits.RotateLeft32(uint32(arg2.I32), int(arg1.I32)))))
        case *core.I32RotrExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(int32(bits.RotateLeft32(uint32(arg2.I32), int(-arg1.I32)))))
        case *core.I32AddExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i32(arg1.I32 + arg2.I32))
        case *core.I32SubExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(RuntimeValue{
                Kind: RuntimeValueI32,
                I32: arg2.I32 - arg1.I32,
            })
        case *core.I64LtsExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if arg2.I64 < arg1.I64 {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I64GtsExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if arg2.I64 > arg1.I64 {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I64GtuExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if uint64(arg2.I64) > uint64(arg1.I64) {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I32GtsExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if arg2.I32 > arg1.I32 {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I32GtuExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if uint32(arg2.I32) > uint32(arg1.I32) {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I32GesExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if arg2.I32 >= arg1.I32 {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I32GeuExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            value := 0
            if uint32(arg2.I32) >= uint32(arg1.I32) {
                value = 1
            }
            stack.Push(i32(int32(value)))
        case *core.I64SubExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i64(arg2.I64 - arg1.I64))
        case *core.I64AddExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i64(arg2.I64 + arg1.I64))
        case *core.I64MulExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            stack.Push(i64(arg1.I64 * arg2.I64))
        case *core.I64EqExpression:
            arg1 := stack.Pop()
            arg2 := stack.Pop()
            if arg1.I64 == arg2.I64 {
                stack.Push(i32(1))
            } else {
                stack.Push(i32(0))
            }
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
            frame.Locals[expr.Local] = stack.Top()

        case *core.LocalSetExpression:
            expr := current.(*core.LocalSetExpression)

            if len(frame.Locals) <= int(expr.Local) {
                return 0, 0, fmt.Errorf("unable to set local %v when frame has %v locals", expr.Local, len(frame.Locals))
            }
            frame.Locals[expr.Local] = stack.Pop()

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
        case *core.GlobalGetExpression:
            expr := current.(*core.GlobalGetExpression)
            index := expr.Global.Id

            if int(index) >= len(store.Globals) {
                return 0, 0, fmt.Errorf("unable to set global %v when store has %v globals", index, len(store.Globals))
            }

            if !store.Globals[index].Mutable {
                return 0, 0, fmt.Errorf("global %v is not mutable", index)
            }

            stack.Push(store.Globals[index].Value)

        case *core.CallIndirectExpression:
            expr := current.(*core.CallIndirectExpression)
            if int(expr.Table.Id) >= len(store.Tables) {
                return 0, 0, fmt.Errorf("invalid table index %v", expr.Table.Id)
            }

            table := store.Tables[expr.Table.Id]
            index := stack.Pop()

            if index.Kind != RuntimeValueI32 {
                return 0, 0, fmt.Errorf("call indirect stack value must be an i32 but was %v", index)
            }

            if index.I32 < 0 || int(index.I32) >= len(table.Elements) {
                return 0, 0, fmt.Errorf("invalid table element index %v", index)
            }

            element := table.Elements[index.I32]
            switch element.(type) {
                case *core.FunctionIndex:
                    ref := element.(*core.FunctionIndex)
                    functionTypeIndex := frame.Module.GetFunctionSection().GetFunctionType(int(ref.Id))
                    functionType := frame.Module.GetTypeSection().GetFunction(functionTypeIndex.Id)

                    args := stack.PopN(len(functionType.InputTypes))

                    code := frame.Module.GetCodeSection().GetFunction(ref.Id)

                    for _, local := range code.Locals {
                        args = append(args, MakeRuntimeValue(local.Type))
                    }

                    out, err := RunCode(code, Frame{
                        Locals: args,
                        Module: frame.Module,
                    }, functionType, store)

                    if err != nil {
                        return 0, 0, err
                    }

                    stack.PushAll(out)
                default:
                    return 0, 0, fmt.Errorf("unknown element for call indirect %v", reflect.TypeOf(element))
            }

        case *core.CallExpression:
            /* create a new stack frame, pop N values off the stack and put them in the locals of the frame.
             * then invoke the code of the function with the new frame.
             * put the resulting runtime value back on the stack
             */
            expr := current.(*core.CallExpression)

            functionTypeIndex := frame.Module.GetFunctionSection().GetFunctionType(int(expr.Index.Id))
            functionType := frame.Module.GetTypeSection().GetFunction(functionTypeIndex.Id)

            args := stack.PopN(len(functionType.InputTypes))

            code := frame.Module.GetCodeSection().GetFunction(expr.Index.Id)

            for _, local := range code.Locals {
                args = append(args, MakeRuntimeValue(local.Type))
            }

            out, err := RunCode(code, Frame{
                Locals: args,
                Module: frame.Module,
            }, functionType, store)

            if err != nil {
                return 0, 0, err
            }

            stack.PushAll(out)

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
func RunCode(code core.Code, frame Frame, functionType core.WebAssemblyFunction, store *Store) ([]RuntimeValue, error) {
    var stack data.Stack[RuntimeValue]
    var labels data.Stack[int]

    instruction := 0

    for instruction < len(code.Expressions) {
        var branch int
        var err error
        instruction, branch, err = Execute(&stack, &labels, code.Expressions, instruction, frame, store)
        if err != nil {
            return nil, err
        }
        if branch == ReturnLabel {
            // FIXME: return the number of values as declared by the function (result ...)
            return stack.PopN(len(functionType.OutputTypes)), nil
        }
        /* if we a branch was executed to label 0, then the 'branch' variable will be equal to 1,
         * which has the same meaning as the end of the function
         */
        if branch == 1 {
            return []RuntimeValue{stack.Pop()}, nil
        }
        if branch != 0 {
            return nil, fmt.Errorf("Branch to non-existent block: %v", branch)
        }
    }

    if stack.Size() != len(functionType.OutputTypes) {
        return nil, fmt.Errorf("too many values still on the stack %v, but expected %v", stack.Size(), len(functionType.OutputTypes))
    }

    return stack.ToArray(), nil
}

/* invoke an exported function in the given module */
func Invoke(module core.WebAssemblyModule, store *Store, name string, args []RuntimeValue) ([]RuntimeValue, error) {
    kind := module.GetExportSection().FindExportByName(name)
    if kind == nil {
        return nil, fmt.Errorf("no such exported function '%v'", name)
    }

    function, ok := kind.(*core.FunctionIndex)
    if ok {
        code := module.GetCodeSection().GetFunction(function.Id)
        functionTypeIndex := module.GetFunctionSection().GetFunctionType(int(function.Id))

        type_ := module.GetTypeSection().GetFunction(functionTypeIndex.Id)

        for _, local := range code.Locals {
            args = append(args, MakeRuntimeValue(local.Type))
        }

        frame := Frame{
            Locals: args,
            Module: module,
        }

        return RunCode(code, frame, type_, store)
    } else {
        return nil, fmt.Errorf("no such exported function '%v'", name)
    }
   
    return nil, fmt.Errorf("shouldnt get here")
}

func cleanName(name string) string {
    return strings.Trim(name, "\"")
}

/* handle wast-style (assert_return ...) */
func AssertReturn(module core.WebAssemblyModule, assert sexp.SExpression, store *Store) error {
    what := assert.Children[0]
    if what.Name == "invoke" {

        functionName := cleanName(what.Children[0].Value)

        var args []RuntimeValue
        for _, arg := range what.Children[1:] {
            expressions := core.MakeExpressions(module, nil, data.Stack[string]{}, arg)
            nextArg, err := EvaluateOne(expressions[0])
            if err != nil {
                return err
            }

            args = append(args, nextArg)
        }

        // store := InitializeStore(module)

        result, err := Invoke(module, store, functionName, args)
        if err != nil {
            return err
        }

        if len(assert.Children) == 2 {
            expressions := core.MakeExpressions(module, nil, data.Stack[string]{}, assert.Children[1])
            expected, err := EvaluateOne(expressions[0])
            if err != nil {
                return err
            } else {
                fail := false

                if len(result) != 1 || result[0] != expected {

                    fail = true

                    /* special-case handling for nan, inf */
                    if result[0].Kind == expected.Kind {
                        switch result[0].Kind {
                            case RuntimeValueF32:
                                if math.IsNaN(float64(result[0].F32)) && math.IsNaN(float64(expected.F32)) {
                                    fail = false
                                }
                            case RuntimeValueF64:
                                if math.IsNaN(result[0].F64) && math.IsNaN(expected.F64) {
                                    fail = false
                                }
                        }
                    }


                    if fail {
                        return fmt.Errorf("result=%v expected=%v", result, expected)
                    }
                }
            }
        }
    }

    return nil
}
