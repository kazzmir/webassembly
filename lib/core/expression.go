package core

import (
    "strings"
    "fmt"
    "bytes"
    "log"
)

type Local struct {
    Count uint32
    Type ValueType
}

type Code struct {
    Locals []Local
    Expressions []Expression
}

type Stack[T any] struct {
    Values []T
}

func (stack *Stack[T]) Size() int {
    return len(stack.Values)
}

func (stack *Stack[T]) Get(depth uint32) T {
    return stack.Values[len(stack.Values) - int(depth) - 1]
}

func (stack *Stack[T]) Push(value T){
    stack.Values = append(stack.Values, value)
}

func (stack *Stack[T]) Pop() T {
    t := stack.Values[len(stack.Values)-1]
    stack.Values = stack.Values[0:len(stack.Values)-1]
    return t
}

func (code *Code) ConvertToWat(indents string) string {
    var out strings.Builder

    var labelStack Stack[int]

    if len(code.Locals) > 0 {
        out.WriteString(indents)
        out.WriteString("(local")
        for i, local := range code.Locals {
            out.WriteByte(' ')
            for x := 0; x < int(local.Count); x++ {
                out.WriteString(local.Type.ConvertToWat(indents))
                if x < int(local.Count) - 1 {
                    out.WriteByte(' ')
                }
            }
            if i < len(code.Locals) - 1 {
                out.WriteByte(' ')
            }
        }
        out.WriteByte(')')
        out.WriteByte('\n')
    }

    for i, expression := range code.Expressions {
        out.WriteString(indents)
        out.WriteString(expression.ConvertToWat(labelStack, indents))
        if i < len(code.Expressions) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (code *Code) AddLocal(count uint32, type_ ValueType){
    code.Locals = append(code.Locals, Local{Count: count, Type: type_})
}

func (code *Code) SetExpressions(expressions []Expression){
    code.Expressions = expressions
}

type Expression interface {
    ConvertToWat(Stack[int], string) string
}

type CallIndirectExpression struct {
    Index *TypeIndex
    Table *TableIndex
}

func (call *CallIndirectExpression) ConvertToWat(labels Stack[int], indents string) string {
    /* FIXME: what to do with the Table field? */
    return fmt.Sprintf("call_indirect (type %v)", call.Index.Id)
}

type CallExpression struct {
    Index *FunctionIndex
}

func (call *CallExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("call %v", call.Index.Id)
}

type BranchIfExpression struct {
    Label uint32
}

func (expr *BranchIfExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("br_if %v (;@%v;)", expr.Label, labels.Get(expr.Label))
}

type BranchExpression struct {
    Label uint32
}

func (expr *BranchExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("br %v (;@%v;)", expr.Label, labels.Get(expr.Label))
}

type RefFuncExpression struct {
    Function *FunctionIndex
}

func (expr *RefFuncExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("ref.func %v", expr.Function.Id)
}

type I32ConstExpression struct {
    N int32
}

func (expr *I32ConstExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("i32.const %v", expr.N)
}

type I64ConstExpression struct {
    N int32
}

func (expr *I64ConstExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("i64.const %v", expr.N)
}

type I32AddExpression struct {
}

func (expr *I32AddExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.add"
}

type I32LoadExpression struct {
    Memory MemoryArgument
}

func (expr *I32LoadExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.load"
}

type I32EqExpression struct {
}

func (expr *I32EqExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.eq"
}

type I32DivSignedExpression struct {
}

func (expr *I32DivSignedExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.div_s"
}

type I32MulExpression struct {
}

func (expr *I32MulExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.mul"
}

type LocalGetExpression struct {
    Local uint32
}

func (expr *LocalGetExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("local.get %v", expr.Local)
}

type DropExpression struct {
}

func (expr *DropExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "drop"
}

type I32CtzExpression struct {
}

func (expr *I32CtzExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i32.ctz"
}

type I64CtzExpression struct {
}

func (expr *I64CtzExpression) ConvertToWat(labels Stack[int], indents string) string {
    return "i64.ctz"
}

type LocalSetExpression struct {
    Local uint32
}

func (expr *LocalSetExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("local.set %v", expr.Local)
}

type GlobalGetExpression struct {
    Global *GlobalIndex
}

func (expr *GlobalGetExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("global.get %v", expr.Global.Id)
}

type GlobalSetExpression struct {
    Global *GlobalIndex
}

func (expr *GlobalSetExpression) ConvertToWat(labels Stack[int], indents string) string {
    return fmt.Sprintf("global.set %v", expr.Global.Id)
}

type BlockKind int
const (
    BlockKindBlock = iota
    BlockKindLoop
    BlockKindIf
)

type BlockExpression struct {
    Expression
    Instructions []Expression
    Kind BlockKind
}

func (block *BlockExpression) ConvertToWat(labels Stack[int], indents string) string {
    labels.Push(labels.Size()+1)
    defer labels.Pop()

    labelNumber := labels.Size()

    var out strings.Builder

    switch block.Kind {
        case BlockKindBlock:
            out.WriteString(fmt.Sprintf("block ;; label = @%v\n", labelNumber))
        case BlockKindLoop:
            out.WriteString(fmt.Sprintf("loop ;; label = @%v\n", labelNumber))
        case BlockKindIf:
            out.WriteString(fmt.Sprintf("if ;; label = @%v\n", labelNumber))
    }

    for _, expression := range block.Instructions {
        out.WriteString(indents + "  ")
        out.WriteString(expression.ConvertToWat(labels, indents + "  "))
        out.WriteByte('\n')
    }
    out.WriteString(indents)
    out.WriteString("end")

    return out.String()
}

type ExpressionSequenceEnd uint32

const (
    SequenceIf ExpressionSequenceEnd = iota
    SequenceEnd ExpressionSequenceEnd = iota
)

const (
    InstructionEnd = 0x0b
)

func ReadBlockInstruction(reader *ByteReader, readingIf bool) (BlockExpression, ExpressionSequenceEnd, error) {
    blockType, err := reader.ReadByte()
    if err != nil {
        return BlockExpression{}, 0, fmt.Errorf("Could not read block type: %v", err)
    }

    if blockType == 0x40 {
    } else {
        /* Read the type from the byte we just read */
        valueType, err := ReadValueType(NewByteReader(bytes.NewReader([]byte{blockType})))
        if err != nil {
            return BlockExpression{}, 0, fmt.Errorf("Unable to read block type: %v", err)
        }
        _ = valueType
    }

    instructions, end, err := ReadExpressionSequence(reader, readingIf)
    if err != nil {
        return BlockExpression{}, 0, fmt.Errorf("Unable to read block instructions: %v", err)
    }

    return BlockExpression{Instructions: instructions}, end, nil
}

/* Read a sequence of instructions. If 'readingIf' is true then we are inside an
 * if-expression so the sequence may end with 0x05, in which case it would
 * be followed by an else sequence of instructions.
 */
func ReadExpressionSequence(reader *ByteReader, readingIf bool) ([]Expression, ExpressionSequenceEnd, error) {

    debug := false
    var sequence []Expression

    count := 0
    for {
        instruction, err := reader.ReadByte()
        if err != nil {
            return nil, 0, fmt.Errorf("Could not read instruction: %v", err)
        }

        if debug {
            log.Printf("Instruction %v: 0x%x\n", count, instruction)
        }

        switch instruction {
            /* unreachable */
            case 0x00: break

            /* nop */
            case 0x01: break

            /* block */
            case 0x02:
                block, _, err := ReadBlockInstruction(reader, false)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read block instruction %v: %v", count, err)
                }

                block.Kind = BlockKindBlock

                sequence = append(sequence, &block)

            /* loop */
            case 0x03:
                loop, _, err := ReadBlockInstruction(reader, false)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read block instruction %v: %v", count, err)
                }

                loop.Kind = BlockKindLoop

                sequence = append(sequence, &loop)

            /* if */
            case 0x04:
                ifBlock, end, err := ReadBlockInstruction(reader, true)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read if block instruction %v: %v", count, err)
                }

                ifBlock.Kind = BlockKindIf

                if end == SequenceIf {
                    elseExpression, _, err := ReadExpressionSequence(reader, false)
                    if err != nil {
                        return nil, 0, fmt.Errorf("Could not read else expressions in if block at instruction %v: %v", count, err)
                    }

                    _ = elseExpression
                }

                sequence = append(sequence, &ifBlock)

            /* else */
            case 0x05:
                if !readingIf {
                    return nil, 0, fmt.Errorf("Read an else bytecode (0x5) outside of an if block at instruction %v", count)
                }

                return nil, SequenceIf, nil

            /* call */
            case 0x10:
                index, err := ReadFunctionIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read function index for call instruction %v: %v", count, err)
                }

                sequence = append(sequence, &CallExpression{Index: index})

            /* call_indirect */
            case 0x11:
                index, err := ReadTypeIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read type index for call_indirect instruction %v: %v", count, err)
                }

                table, err := ReadTableIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read table index for call_indirect instruction %v: %v", count, err)
                }

                sequence = append(sequence, &CallIndirectExpression{Index: index, Table: table})

            /* termination of a block / instruction sequence */
            case 0xb:
                if debug {
                    log.Printf("Read %v instructions\n", count+1)
                }
                return sequence, SequenceEnd, nil

            /* br */
            case 0xc:
                label, err := ReadU32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read label index for br at instruction %v: %v", count, err)
                }

                sequence = append(sequence, &BranchExpression{Label: label})

            /* br_if */
            case 0xd:
                label, err := ReadU32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read label index for br_if at instruction %v: %v", count, err)
                }

                sequence = append(sequence, &BranchIfExpression{Label: label})

            /* br_table */
            case 0xe:
                labels, err := ReadU32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read labels length for br_table instruction %v: %v", count, err)
                }

                var i uint32
                for i = 0; i < labels; i++ {
                    index, err := ReadU32(reader)
                    if err != nil {
                        return nil, 0, fmt.Errorf("Could not read label index %v for br_table instruction %v: %v", i, count, err)
                    }

                    _ = index
                }

                lastIndex, err := ReadU32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read the last label index for br_table instruction %v: %v", count, err)
                }

                _ = lastIndex

            /* return */
            case 0xf:
                break

            /* drop */
            case 0x1a:
                break

            /* select */
            case 0x1b:
                break

            /* local.get */
            case 0x20:
                local, err := ReadLocalIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read local index instruction %v: %v", count, err)
                }

                sequence = append(sequence, &LocalGetExpression{Local: local})

            /* local.set */
            case 0x21:
                local, err := ReadLocalIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read local index instruction %v: %v", count, err)
                }

                sequence = append(sequence, &LocalSetExpression{Local: local})

            /* local.tee */
            case 0x22:
                local, err := ReadLocalIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read local index instruction %v: %v", count, err)
                }

                _ = local

            /* global.get */
            case 0x23:
                global, err := ReadGlobalIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read global index instruction %v: %v", count, err)
                }

                sequence = append(sequence, &GlobalGetExpression{Global: global})

            /* global.set */
            case 0x24:
                global, err := ReadGlobalIndex(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read global index instruction %v: %v", count, err)
                }

                sequence = append(sequence, &GlobalSetExpression{Global: global})

                /* i32.load */
            case 0x28,
                 /* i64.load */
                 0x29,
                 /* f32.load */
                 0x2a,
                 /* f64.load */
                 0x2b,
                 /* i32.load8_s */
                 0x2c,
                 /* i32.load8_u */
                 0x2d,
                 /* i32.load16_s */
                 0x2e,
                 /* i32.load16_u */
                 0x2f,
                 /* i64.load8_s */
                 0x30,
                 /* i64.load8_u */
                 0x31,
                 /* i64.load16_s */
                 0x32,
                 /* i64.load16_u */
                 0x33,
                 /* i64.load32_s */
                 0x34,
                 /* i64.load32_u */
                 0x35,
                 /* i32.store */
                 0x36,
                 /* i64.store */
                 0x37,
                 /* f32.store */
                 0x38,
                 /* f64.store */
                 0x39,
                 /* i32.store8 */
                 0x3a,
                 /* i32.store16 */
                 0x3b,
                 /* i64.store8 */
                 0x3c,
                 /* i64.store16 */
                 0x3d,
                 /* i64.store32 */
                 0x3e:

                memory, err := ReadMemoryArgument(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read memory argument for instruction %v: %v", count, err)
                }

                switch instruction {
                    case 0x28:
                        sequence = append(sequence, &I32LoadExpression{Memory: memory})
                }

            /* memory.size */
            case 0x3f,
                /* memory.grow */
                0x40:

                name := "memory.size"
                if instruction == 0x40 {
                    name = "memory.grow"
                }

                zero, err := reader.ReadByte()
                if err != nil {
                    return nil, 0, fmt.Errorf("Could not read extra zero-byte for %s instruction %v: %v", name, count, err)
                }

                if zero != 0 {
                    return nil, 0, fmt.Errorf("Expected byte following %s instruction %v to be 0 but got %v", name, count, zero)
                }

            /* i32.const n */
            case 0x41:
                i32, err := ReadS32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Unable to read i32 value at instruction %v: %v", count, err)
                }

                sequence = append(sequence, &I32ConstExpression{N: i32})

            /* i64.const n */
            case 0x42:
                i64, err := ReadS64(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Unable to read i64 value at instruction %v: %v", count, err)
                }

                _ = i64

            /* f32.const */
            case 0x43:
                f32, err := ReadFloat32(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Unable to read f32 value at instruction %v: %v", count, err)
                }

                _ = f32

            /* f64.const */
            case 0x44:
                f64, err := ReadFloat64(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Unable to read f64 value at instruction %v: %v", count, err)
                }

                _ = f64

            /* No-argument instructions */

                /* i32.eqz */
            case 0x45:
                /* i32.eq */
                break
            case 0x46:
                sequence = append(sequence, &I32EqExpression{})

                /* i32.ne */
            case 0x47,
                /* i32.lt_s */
                0x48,
                /* i32.lt_u */
                0x49,
                /* i32.gt_s */
                0x4a,
                /* i32.gt_u */
                0x4b,
                /* i32.le_s */
                0x4c,
                /* i32.le_u */
                0x4d,
                /* i32.ge_s */
                0x4e,
                /* i32.ge_u */
                0x4f,
                /* i64.eqz */
                0x50,
                /* i64.eq */
                0x51,
                /* i64.ne */
                0x52,
                /* i64.lt_s */
                0x53,
                /* i64.lt_u */
                0x54,
                /* i64.gt_s */
                0x55,
                /* i64.gt_u */
                0x56,
                /* i64.le_s */
                0x57,
                /* i64.le_u */
                0x58,
                /* i64.ge_s */
                0x59,
                /* i64.ge_u */
                0x5a,
                /* f32.eq */
                0x5b,
                /* f32.ne */
                0x5c,
                /* f32.lt */
                0x5d,
                /* f32.gt */
                0x5e,
                /* f32.le */
                0x5f,
                /* f32.ge */
                0x60,
                /* f64.eq */
                0x61,
                /* f64.ne */
                0x62,
                /* f64.lt */
                0x63,
                /* f64.gt */
                0x64,
                /* f64.le */
                0x65,
                /* f64.ge */
                0x66,
                /* i32.clz */
                0x67,
                /* i32.ctz */
                0x68,
                /* i32.popcnt */
                0x69:
                    break
                /* i32.add */
            case 0x6a:
                sequence = append(sequence, &I32AddExpression{})

                /* i32.sub */
            case 0x6b:
                break
                /* i32.mul */
            case 0x6c:
                sequence = append(sequence, &I32MulExpression{})
                /* i32.div_s */
            case 0x6d:
                sequence = append(sequence, &I32DivSignedExpression{})

                /* i32.div_u */
            case 0x6e,
                /* i32.rem_s */
                0x6f,
                /* i32.rem_u */
                0x70,
                /* i32.and */
                0x71,
                /* i32.or */
                0x72,
                /* i32.xor */
                0x73,
                /* i32.shl */
                0x74,
                /* i32.shr_s */
                0x75,
                /* i32.shr_u */
                0x76,
                /* i32.rotl */
                0x77,
                /* i32.rotr */
                0x78,
                /* i64.clz */
                0x79,
                /* i64.ctz */
                0x7a,
                /* i64.popcnt */
                0x7b,
                /* i64.add */
                0x7c,
                /* i64.sub */
                0x7d,
                /* i64.mul */
                0x7e,
                /* i64.div_s */
                0x7f,
                /* i64.div_u */
                0x80,
                /* i64.rem_s */
                0x81,
                /* i64.rem_u */
                0x82,
                /* i64.and */
                0x83,
                /* i64.or */
                0x84,
                /* i64.xor */
                0x85,
                /* i64.shl */
                0x86,
                /* i64.shr_s */
                0x87,
                /* i64.shr_u */
                0x88,
                /* i64.rotl */
                0x89,
                /* i64.rotr */
                0x8a,
                /* f32.abs */
                0x8b,
                /* f32.neg */
                0x8c,
                /* f32.ceil */
                0x8d,
                /* f32.floor */
                0x8e,
                /* f32.trunc */
                0x8f,
                /* f32.nearest */
                0x90,
                /* f32.sqrt */
                0x91,
                /* f32.add */
                0x92,
                /* f32.sub */
                0x93,
                /* f32.mul */
                0x94,
                /* f32.div */
                0x95,
                /* f32.min */
                0x96,
                /* f32.max */
                0x97,
                /* f32.copysign */
                0x98,
                /* f64.abs */
                0x99,
                /* f64.neg */
                0x9a,
                /* f64.ceil */
                0x9b,
                /* f64.floor */
                0x9c,
                /* f64.trunc */
                0x9d,
                /* f64.nearest */
                0x9e,
                /* f64.sqrt */
                0x9f,
                /* f64.add */
                0xa0,
                /* f64.sub */
                0xa1,
                /* f64.mul */
                0xa2,
                /* f64.div */
                0xa3,
                /* f64.min */
                0xa4,
                /* f64.max */
                0xa5,
                /* f64.copysign */
                0xa6,
                /* i32.wrap_i64 */
                0xa7,
                /* i32.trunc_f32_s */
                0xa8,
                /* i32.trunc_f32_u */
                0xa9,
                /* i32.trunc_f64_s */
                0xaa,
                /* i32.trunc_f64_u */
                0xab,
                /* i64.extend_i32_s */
                0xac,
                /* i64.extend_i32_u */
                0xad,
                /* i64.trunc_f32_s */
                0xae,
                /* i64.trunc_f32_u */
                0xaf,
                /* i64.trunc_f64_s */
                0xb0,
                /* i64.trunc_f64_u */
                0xb1,
                /* f32.convert_i32_s */
                0xb2,
                /* f32.convert_i32_u */
                0xb3,
                /* f32.convert_i64_s */
                0xb4,
                /* f32.convert_i64_u */
                0xb5,
                /* f32.demote_f64 */
                0xb6,
                /* f64.convert_i32_s */
                0xb7,
                /* f64.convert_i32_u */
                0xb8,
                /* f64.convert_i64_s */
                0xb9,
                /* f64.convert_i64_u */
                0xba,
                /* f64.promote_f32 */
                0xbb,
                /* i32.reinterpret_f32 */
                0xbc,
                /* i64.reinterpret_f64 */
                0xbd,
                /* f32.reinterpret_i32 */
                0xbe,
                /* f64.reinterpret_i64 */
                0xbf:

                break

            default:
                return nil, 0, fmt.Errorf("Unimplemented instruction 0x%x", instruction)
        }

        count += 1
    }
}

func ReadCode(reader *ByteReader) (Code, error) {
    locals, err := ReadU32(reader)
    if err != nil {
        return Code{}, fmt.Errorf("Could not read locals: %v", err)
    }
    debug := false

    if debug {
        log.Printf("Read code locals %v\n", locals)
    }

    var code Code

    var i uint32
    for i = 0; i < locals; i++ {
        count, err := ReadU32(reader)
        if err != nil {
            return Code{}, fmt.Errorf("Could not read local count for %v: %v", i, err)
        }

        valueType, err := ReadValueType(reader)
        if err != nil {
            return Code{}, fmt.Errorf("Could not read type of local for %v: %v", i, err)
        }

        if debug {
            log.Printf("Local %v; count=%v type=0x%x\n", i, count, valueType)
        }

        code.AddLocal(count, valueType)
    }

    expressions, _, err := ReadExpressionSequence(reader, false)
    if err != nil {
        return Code{}, fmt.Errorf("Could not read expressions: %v", err)
    }

    code.SetExpressions(expressions)

    return code, nil
}
