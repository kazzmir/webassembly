package core

import (
    "strings"
    "fmt"
    "bytes"
    "log"

    "github.com/kazzmir/webassembly/lib/data"
)

type Local struct {
    Count uint32
    Name string
    Type ValueType
}

type Code struct {
    Locals []Local
    Expressions []Expression
}

func (code *Code) LookupLocal(name string) (int, bool) {
    for i := 0; i < len(code.Locals); i++ {
        if code.Locals[i].Name == name {
            return i, true
        }
    }

    return 0, false
}

func (code *Code) ConvertToWat(indents string) string {
    var out strings.Builder

    var labelStack data.Stack[int]
    /* the function implicitly creates a label */
    labelStack.Push(0)

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
    ConvertToWat(data.Stack[int], string) string
}

type CallIndirectExpression struct {
    Index *TypeIndex
    Table *TableIndex
}

func (call *CallIndirectExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    /* FIXME: what to do with the Table field? */
    return fmt.Sprintf("call_indirect (type %v)", call.Index.Id)
}

type CallExpression struct {
    Index *FunctionIndex
}

func (call *CallExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("call %v", call.Index.Id)
}

type BranchTableExpression struct {
    Labels []uint32
}

func (expr *BranchTableExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    var out strings.Builder

    out.WriteString("br_table")
    for _, label := range expr.Labels {
        out.WriteString(fmt.Sprintf(" %v (;@%v;)", label, labels.Get(label)))
    }

    return out.String()
}

type BranchIfExpression struct {
    Label uint32
}

func (expr *BranchIfExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    if int(expr.Label) > labels.Size() {
        return fmt.Sprintf("invalid label %v", expr.Label)
    }

    return fmt.Sprintf("br_if %v (;@%v;)", expr.Label, labels.Get(expr.Label))
}

type BranchExpression struct {
    Label uint32
}

func (expr *BranchExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("br %v (;@%v;)", expr.Label, labels.Get(expr.Label))
}

type RefFuncExpression struct {
    Function *FunctionIndex
}

func (expr *RefFuncExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("ref.func %v", expr.Function.Id)
}

type F32NegExpression struct {
}

func (expr *F32NegExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "f32.neg"
}

type F64NegExpression struct {
}

func (expr *F64NegExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "f64.neg"
}

type F32ConstExpression struct {
    N float32
}

func (expr *F32ConstExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.const %v", expr.N)
}

type F32GtExpression struct {
}

func (expr *F32GtExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.gt")
}

type F32EqExpression struct {
}

func (expr *F32EqExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.eq")
}

type F32LtExpression struct {
}

func (expr *F32LtExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.lt")
}

type F32AddExpression struct {
}

func (expr *F32AddExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.add")
}

type F32SqrtExpression struct {
}

func (expr *F32SqrtExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.sqrt")
}

type F32SubExpression struct {
}

func (expr *F32SubExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.sub")
}

type F32DivExpression struct {
}

func (expr *F32DivExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.div")
}

type F64ConstExpression struct {
    N float64
}

func (expr *F64ConstExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.const %v", expr.N)
}

type F32LoadExpression struct {
}

func (expr *F32LoadExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.load")
}

type F64LoadExpression struct {
}

func (expr *F64LoadExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.load")
}

type F32ReinterpretI32Expression struct {
}

func (expr *F32ReinterpretI32Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.reinterpret_i32")
}

type F64ReinterpretI64Expression struct {
}

func (expr *F64ReinterpretI64Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.reinterpret_i64")
}

type F64StoreExpression struct {
}

func (expr *F64StoreExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.store")
}

type F64AddExpression struct {
}

func (expr *F64AddExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.add")
}

type F64LeExpression struct {
}

func (expr *F64LeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.le")
}

type F64NeExpression struct {
}

func (expr *F64NeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.ne")
}

type F64ConvertI64uExpression struct {
}

func (expr *F64ConvertI64uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.convert_i64_u")
}

type F64PromoteF32Expression struct {
}

func (expr *F64PromoteF32Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.promote_f32")
}

type F64ConvertI32uExpression struct {
}

func (expr *F64ConvertI32uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.convert_i32_u")
}

type F64ConvertI32sExpression struct {
}

func (expr *F64ConvertI32sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.convert_i32_s")
}

type F64SubExpression struct {
}

func (expr *F64SubExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.sub")
}

type F64MulExpression struct {
}

func (expr *F64MulExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.mul")
}

type F64DivExpression struct {
}

func (expr *F64DivExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.div")
}

type F64CopySignExpression struct {
}

func (expr *F64CopySignExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.copysign")
}

type F64EqExpression struct {
}

func (expr *F64EqExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.eq")
}

type F64LtExpression struct {
}

func (expr *F64LtExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.lt")
}

type F64GtExpression struct {
}

func (expr *F64GtExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.gt")
}

type F64MinExpression struct {
}

func (expr *F64MinExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.min")
}

type F64MaxExpression struct {
}

func (expr *F64MaxExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.max")
}

type F64GeExpression struct {
}

func (expr *F64GeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f64.ge")
}

type F32MulExpression struct {
}

func (expr *F32MulExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.mul")
}

type F32CopySignExpression struct {
}

func (expr *F32CopySignExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.copysign")
}

type F32LeExpression struct {
}

func (expr *F32LeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.le")
}

type F32GeExpression struct {
}

func (expr *F32GeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.ge")
}

type F32MinExpression struct {
}

func (expr *F32MinExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.min")
}

type F32MaxExpression struct {
}

func (expr *F32MaxExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.max")
}

type F32StoreExpression struct {
}

func (expr *F32StoreExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.store")
}

type F32NeExpression struct {
}

func (expr *F32NeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("f32.ne")
}

type RefFuncNullExpression struct {
}

func (expr *RefFuncNullExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("ref.null func")
}

type RefExternNullExpression struct {
}

func (expr *RefExternNullExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("ref.null externref")
}

type RefExternExpression struct {
    Id uint32
}

func (expr *RefExternExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf(fmt.Sprintf("ref.extern %v", expr.Id))
}

type UnreachableExpression struct {
}

func (expr *UnreachableExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("unreachable")
}

type I32WrapI64Expression struct {
}

func (expr *I32WrapI64Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("i32.wrap_i64")
}

type I32LtuExpression struct {
}

func (expr *I32LtuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("i32.lt_u")
}

type I32LtsExpression struct {
}

func (expr *I32LtsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("i32.lt_s")
}

type I32ConstExpression struct {
    N int32
}

func (expr *I32ConstExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("i32.const %v", expr.N)
}

type I32Extend8sExpression struct {
}

func (expr *I32Extend8sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.extend8_s"
}

type I32Extend16sExpression struct {
}

func (expr *I32Extend16sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.extend16_s"
}

type I64ExtendI32sExpression struct {
}

func (expr *I64ExtendI32sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.extend_i32_s"
}

type I64LtsExpression struct {
}

func (expr *I64LtsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.lt_s"
}

type I64GtsExpression struct {
}

func (expr *I64GtsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.gt_s"
}

type I64GtuExpression struct {
}

func (expr *I64GtuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.gt_u"
}

type I32GtsExpression struct {
}

func (expr *I32GtsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.gt_s"
}

type I32GtuExpression struct {
}

func (expr *I32GtuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.gt_u"
}

type I32GesExpression struct {
}

func (expr *I32GesExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.ge_s"
}

type I32GeuExpression struct {
}

func (expr *I32GeuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.ge_u"
}

type I64AddExpression struct {
}

func (expr *I64AddExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.add"
}

type I64ConstExpression struct {
    N int64
}

func (expr *I64ConstExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("i64.const %v", expr.N)
}

type I32AddExpression struct {
}

func (expr *I32AddExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.add"
}

type I64SubExpression struct {
}

func (expr *I64SubExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.sub"
}

type I32SubExpression struct {
}

func (expr *I32SubExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.sub"
}

type I64ExtendI32uExpression struct {
}

func (expr *I64ExtendI32uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.extend_i32_u"
}

type I64LtuExpression struct {
}

func (expr *I64LtuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.lt_u"
}

type I32Load16sExpression struct {
}

func (expr *I32Load16sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.load16_s"
}

type I32Load16uExpression struct {
}

func (expr *I32Load16uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.load16_u"
}

type I64Load16sExpression struct {
}

func (expr *I64Load16sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load16_s"
}

type I64Load16uExpression struct {
}

func (expr *I64Load16uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load16_u"
}

type I64Load32sExpression struct {
}

func (expr *I64Load32sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load32_s"
}

type I64Load32uExpression struct {
}

func (expr *I64Load32uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load32_u"
}

type I64LoadExpression struct {
}

func (expr *I64LoadExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load"
}

type I32ReinterpretF32Expression struct {
}

func (expr *I32ReinterpretF32Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.reinterpret_f32"
}

type I64ReinterpretF64Expression struct {
}

func (expr *I64ReinterpretF64Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.reinterpret_f64"
}

/* FIXME: generalize these load expression types */
type I32Load8sExpression struct {
    Memory MemoryArgument
}

func (expr *I32Load8sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.load8_s"
}

type I32Load8uExpression struct {
}

func (expr *I32Load8uExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.load8_u"
}

type I64Load8sExpression struct {
    Memory MemoryArgument
}

func (expr *I64Load8sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.load8_s"
}

type I64DivsExpression struct {
}

func (expr *I64DivsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.div_s"
}

type I64DivuExpression struct {
}

func (expr *I64DivuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.div_u"
}

type I64RemsExpression struct {
}

func (expr *I64RemsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.rem_s"
}

type I64RemuExpression struct {
}

func (expr *I64RemuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.rem_u"
}

type I64AndExpression struct {
}

func (expr *I64AndExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.and"
}

type I64OrExpression struct {
}

func (expr *I64OrExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.or"
}

type I64XOrExpression struct {
}

func (expr *I64XOrExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.xor"
}

type I64ShlExpression struct {
}

func (expr *I64ShlExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.shl"
}

type I64ShruExpression struct {
}

func (expr *I64ShruExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.shr_u"
}

type I64ShrsExpression struct {
}

func (expr *I64ShrsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.shr_s"
}

type I64NeExpression struct {
}

func (expr *I64NeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.ne"
}

type I64LesExpression struct {
}

func (expr *I64LesExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.le_s"
}

type I64GesExpression struct {
}

func (expr *I64GesExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.ge_s"
}

type I64GeuExpression struct {
}

func (expr *I64GeuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.ge_u"
}

type I32StoreExpression struct {
}

func (expr *I32StoreExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.store"
}

type I64StoreExpression struct {
}

func (expr *I64StoreExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.store"
}

type I64Store8Expression struct {
}

func (expr *I64Store8Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.store8"
}

type I64Store32Expression struct {
}

func (expr *I64Store32Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.store32"
}

type I64Store16Expression struct {
}

func (expr *I64Store16Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.store16"
}

type I32Store8Expression struct {
}

func (expr *I32Store8Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.store8"
}

type I32Store16Expression struct {
}

func (expr *I32Store16Expression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.store16"
}

type I32LoadExpression struct {
    Memory MemoryArgument
}

func (expr *I32LoadExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.load"
}

type I32EqzExpression struct {
}

func (expr *I32EqzExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.eqz"
}

type I32LeuExpression struct {
}

func (expr *I32LeuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.le_u"
}

type I32LesExpression struct {
}

func (expr *I32LesExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.le_s"
}

type I32DivsExpression struct {
}

func (expr *I32DivsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.div_s"
}

type I32DivuExpression struct {
}

func (expr *I32DivuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.div_u"
}

type I32RemsExpression struct {
}

func (expr *I32RemsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.rem_s"
}

type I32RemuExpression struct {
}

func (expr *I32RemuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.rem_u"
}

type I32AndExpression struct {
}

func (expr *I32AndExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.and"
}

type I32OrExpression struct {
}

func (expr *I32OrExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.or"
}

type I32XOrExpression struct {
}

func (expr *I32XOrExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.xor"
}

type I32ShlExpression struct {
}

func (expr *I32ShlExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.shl"
}

type I32ShlsExpression struct {
}

func (expr *I32ShlsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.shl_s"
}

type I32ShluExpression struct {
}

func (expr *I32ShluExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.shl_u"
}

type I32ShrsExpression struct {
}

func (expr *I32ShrsExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.shr_s"
}

type I32ShruExpression struct {
}

func (expr *I32ShruExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.shr_u"
}

type I32RotlExpression struct {
}

func (expr *I32RotlExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.rotl"
}

type I32RotrExpression struct {
}

func (expr *I32RotrExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.rotr"
}

type I64LeuExpression struct {
}

func (expr *I64LeuExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.le_u"
}

type I32NeExpression struct {
}

func (expr *I32NeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.ne"
}

type I32EqExpression struct {
}

func (expr *I32EqExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.eq"
}

type I64EqExpression struct {
}

func (expr *I64EqExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.eq"
}

type I64EqzExpression struct {
}

func (expr *I64EqzExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.eqz"
}

type I32DivSignedExpression struct {
}

func (expr *I32DivSignedExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.div_s"
}

type I32MulExpression struct {
}

func (expr *I32MulExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.mul"
}

type I64MulExpression struct {
}

func (expr *I64MulExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.mul"
}

type MemoryGrowExpression struct {
}

func (expr *MemoryGrowExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "memory.grow"
}

type LocalGetExpression struct {
    Local uint32
}

func (expr *LocalGetExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("local.get %v", expr.Local)
}

type LocalTeeExpression struct {
    Local uint32
}

func (expr *LocalTeeExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("local.tee %v", expr.Local)
}

type ReturnExpression struct {
}

func (expr *ReturnExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("return")
}

type DropExpression struct {
}

func (expr *DropExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "drop"
}

type I32CtzExpression struct {
}

func (expr *I32CtzExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.ctz"
}

type I32ClzExpression struct {
}

func (expr *I32ClzExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.clz"
}

type I32PopcntExpression struct {
}

func (expr *I32PopcntExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i32.popcnt"
}

type I64CtzExpression struct {
}

func (expr *I64CtzExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.ctz"
}

type I64TruncF64sExpression struct {
}

func (expr *I64TruncF64sExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "i64.trunc_f64_s"
}

type LocalSetExpression struct {
    Local uint32
}

func (expr *LocalSetExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("local.set %v", expr.Local)
}

type GlobalGetExpression struct {
    Global *GlobalIndex
}

func (expr *GlobalGetExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("global.get %v", expr.Global.Id)
}

type GlobalSetExpression struct {
    Global *GlobalIndex
}

func (expr *GlobalSetExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return fmt.Sprintf("global.set %v", expr.Global.Id)
}

type SelectExpression struct {
}

func (expr *SelectExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    return "select"
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
    ElseInstructions []Expression // for if-then-else
    Kind BlockKind
    ExpectedType []ValueType
}

func (block *BlockExpression) ConvertToWat(labels data.Stack[int], indents string) string {
    labels.Push(labels.Size())
    defer labels.Pop()

    labelNumber := labels.Size() - 1

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

                sequence = append(sequence, &F32ConstExpression{N: f32})

            /* f64.const */
            case 0x44:
                f64, err := ReadFloat64(reader)
                if err != nil {
                    return nil, 0, fmt.Errorf("Unable to read f64 value at instruction %v: %v", count, err)
                }

                sequence = append(sequence, &F64ConstExpression{N: f64})

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
