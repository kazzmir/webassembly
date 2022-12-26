package core

import (
    "fmt"
    "strings"
)

type WebAssemblySection interface {
    String() string
    // convert to the .wat text file format
    // https://webassembly.github.io/spec/core/text/index.html
    ConvertToWat(module *WebAssemblyModule, indents string) string
    ToInterface() WebAssemblySection
}

type WebAssemblyStartSection struct {
    Start FunctionIndex
}

func (section *WebAssemblyStartSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyStartSection) String() string {
    return "start section"
}

func (section *WebAssemblyStartSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    return fmt.Sprintf("(start %v)", section.Start.Id)
}

type MemoryMode interface {
}

type MemoryActiveMode struct {
    Memory uint32
    Offset []Expression
}

type MemoryPassiveMode struct {
}

type DataSegment struct {
    Data []byte
    Mode MemoryMode
}

type WebAssemblyDataSection struct {
    Segments []DataSegment
}

func (section *WebAssemblyDataSection) AddData(data []byte, mode MemoryMode){
    section.Segments = append(section.Segments, DataSegment{
        Data: data,
        Mode: mode,
    })
}

func (section *WebAssemblyDataSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }

    return section
}

func (section *WebAssemblyDataSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder

    for i, item := range section.Segments {
        out.WriteString(indents)
        out.WriteString(fmt.Sprintf("(data (;%v;) ", i))

        switch item.Mode.(type) {
            case *MemoryActiveMode:
                active := item.Mode.(*MemoryActiveMode)
                out.WriteByte('(')
                for e, expr := range active.Offset {
                    var label Stack[int]
                    out.WriteString(expr.ConvertToWat(label, ""))
                    if e < len(active.Offset) - 1 {
                        out.WriteByte(' ')
                    }
                }
                out.WriteByte(')')
        }

        out.WriteByte(' ')
        out.WriteByte('"')
        out.Write(item.Data)
        out.WriteByte('"')

        out.WriteByte(')')
        if i < len(section.Segments) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (section *WebAssemblyDataSection) String() string {
    return "data section"
}

type Global struct {
    Global *GlobalType
    Expression []Expression
}

type WebAssemblyGlobalSection struct {
    Globals []Global
}

func (section *WebAssemblyGlobalSection) AddGlobal(global *GlobalType, expression []Expression){
    section.Globals = append(section.Globals, Global{Global: global, Expression: expression})
}

func (section *WebAssemblyGlobalSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyGlobalSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder
    for i, global := range section.Globals {
        out.WriteString(indents)
        out.WriteString("(global")
        _ = global
        out.WriteByte(')')
        if i < len(section.Globals) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (section *WebAssemblyGlobalSection) String() string {
    return "global section"
}

type WebAssemblyMemorySection struct {
}

func (section *WebAssemblyMemorySection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyMemorySection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    return "(memory)"
}

func (section *WebAssemblyMemorySection) String() string {
    return "memory section"
}

type WebAssemblyCustomSection struct {
}

func (section *WebAssemblyCustomSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyCustomSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    return "(custom)"
}

func (section *WebAssemblyCustomSection) String() string {
    return "custom section"
}

type ElementInit struct {
    Type byte
    Inits []Expression
    Mode ElementMode
}

type ElementMode interface {
}

type ElementModeActive struct {
    Table int
    Offset []Expression
}

type WebAssemblyElementSection struct {
    Elements []ElementInit
}

func (section *WebAssemblyElementSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }

    return section
}

func (section *WebAssemblyElementSection) AddFunctionRefInit(functions []*FunctionIndex, expression []Expression){
    var inits []Expression
    for _, function := range functions {
        inits = append(inits, &RefFuncExpression{
            Function: function,
        })
    }
    section.Elements = append(section.Elements, ElementInit{
        Type: RefTypeFunction,
        Inits: inits,
        Mode: &ElementModeActive{
            Table: 0,
            Offset: expression,
        },
    })
}

func (section *WebAssemblyElementSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder
    for i, element := range section.Elements {
        out.WriteString(indents)
        out.WriteString(fmt.Sprintf("(elem (;%v;) ", i))

        active, isActive := element.Mode.(*ElementModeActive)
        if isActive {
            out.WriteString("(")
            var labels Stack[int]
            for _, expr := range active.Offset {
                out.WriteString(expr.ConvertToWat(labels, indents))
            }
            out.WriteString(") ")
        }

        switch element.Type {
            case RefTypeFunction:
                out.WriteString("func ")
        }

        for i, init := range element.Inits {
            // out.WriteString(init.Function.Id)
            refFunc, ok := init.(*RefFuncExpression)
            if ok {
                out.WriteString(fmt.Sprintf("%v", refFunc.Function.Id))
            }
            if i < len(element.Inits) - 1 {
                out.WriteString(" ")
            }
        }

        out.WriteByte(')')
        if i < len(section.Elements) {
            out.WriteByte('\n')
        }
    }
    return out.String()
}

func (section *WebAssemblyElementSection) String() string {
    return "element section"
}

type TableType struct {
    Limit Limit
    RefType byte
}

func (table *TableType) String() string {
    return fmt.Sprintf("reftype=%v min=%v max=%v", table.RefType, table.Limit.Minimum, table.Limit.Maximum)
}

func ReadTableType(reader *ByteReader) (*TableType, error) {
    refType, err := reader.ReadByte()
    if err != nil {
        return nil, fmt.Errorf("Could not read table type: %v", err)
    }

    if refType != RefTypeFunction && refType != RefTypeExtern {
        return nil, fmt.Errorf("Unexpected table ref type %v", refType)
    }

    limit, err := ReadLimit(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read limit for table type: %v", err)
    }

    return &TableType{
        Limit: limit,
        RefType: refType,
    }, nil
}

type WebAssemblyTableSection struct {
    Items []TableType
}

func (section *WebAssemblyTableSection) AddTable(table TableType){
    section.Items = append(section.Items, table)
}

func (section *WebAssemblyTableSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyTableSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder

    for i, item := range section.Items {
        out.WriteString(indents)
        out.WriteString(fmt.Sprintf("(table (;%v;) ", i))
        if item.Limit.HasMaximum {
            out.WriteString(fmt.Sprintf("%v %v ", item.Limit.Minimum, item.Limit.Maximum))
        } else {
            out.WriteString(fmt.Sprintf("%v ", item.Limit.Minimum))
        }

        switch item.RefType {
            case RefTypeFunction:
                out.WriteString("funcref")
            case RefTypeExtern:
                out.WriteString("externref")
        }

        out.WriteByte(')')
        if i < len(section.Items) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (section *WebAssemblyTableSection) String() string {
    return "table section"
}

type WebAssemblyCodeSection struct {
    Code []Code
}

func (section *WebAssemblyCodeSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }

    return section
}

func (section *WebAssemblyCodeSection) AddCode(code Code){
    section.Code = append(section.Code, code)
}

func (section *WebAssemblyCodeSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder
    startIndex := module.GetImportFunctionCount()
    for i, code := range section.Code {
        out.WriteString(indents)
        typeIndex := module.FindFunctionType(i)
        if typeIndex == nil {
            out.WriteString(fmt.Sprintf("unknown function type index for function %v", i))
        } else {
            out.WriteString(fmt.Sprintf("(func (;%v;) (type %v)", i+startIndex, typeIndex.Id))

            function := module.GetFunction(typeIndex.Id)

            if len(function.InputTypes) > 0 {
                out.WriteString(" (param")
                for _, input := range function.InputTypes {
                    out.WriteByte(' ')
                    out.WriteString(input.ConvertToWat(""))
                }
                out.WriteByte(')')
            }


            if len(function.OutputTypes) > 0 {
                out.WriteString(" (result")
                for _, output := range function.OutputTypes {
                    out.WriteByte(' ')
                    out.WriteString(output.ConvertToWat(""))
                }
                out.WriteString(")")
            }
        }
        if len(code.Expressions) > 0 {
            out.WriteByte('\n')
            out.WriteString(code.ConvertToWat(indents + "  "))
        }
        out.WriteString(")\n")
    }
    return out.String()
}

func (section *WebAssemblyCodeSection) String() string {
    return "code section"
}

type ExportSectionItem struct {
    Name string
    Kind Index
}

type WebAssemblyExportSection struct {
    Items []ExportSectionItem
}

func (section *WebAssemblyExportSection) AddExport(name string, kind Index){
    section.Items = append(section.Items, ExportSectionItem{
        Name: name,
        Kind: kind,
    })
}

func (section *WebAssemblyExportSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyExportSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder

    for i, item := range section.Items {
        out.WriteString(indents)
        out.WriteString(fmt.Sprintf("(export \"%v\" ", item.Name))

        found := false
        func_, ok := item.Kind.(*FunctionIndex)
        if ok {
            out.WriteString(fmt.Sprintf("(func %v)", func_.Id))
            found = true
        }

        table, ok := item.Kind.(*TableIndex)
        if ok {
            out.WriteString(fmt.Sprintf("(table %v)", table.Id))
            found = true
        }

        if !found {
            out.WriteString(fmt.Sprintf("unhandled export index %+v", item.Kind))
        }

        out.WriteByte(')')
        if i < len(section.Items) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (section *WebAssemblyExportSection) String() string {
    return "export section"
}

type WebAssemblyFunctionSection struct {
    Functions []*TypeIndex
}

func (section *WebAssemblyFunctionSection) GetFunctionType(index int) *TypeIndex {
    if index >= 0 && index < len(section.Functions) {
        return section.Functions[index]
    }

    return nil
}

func (section *WebAssemblyFunctionSection) AddFunction(index *TypeIndex) uint32 {
    section.Functions = append(section.Functions, index)
    return uint32(len(section.Functions) - 1)
}

func (section *WebAssemblyFunctionSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyFunctionSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    return ""
    // return indents + "(function)"
}

func (section *WebAssemblyFunctionSection) String() string {
    return "function section"
}

type ImportSectionItem struct {
    ModuleName string
    Name string
    Kind Index
}

type WebAssemblyImportSection struct {
    Items []ImportSectionItem
}

func (section *WebAssemblyImportSection) CountFunctions() int {
    count := 0
    for _, item := range section.Items {
        _, ok := item.Kind.(*FunctionImport)
        if ok {
            count += 1
        }
    }

    return count
}

func (section *WebAssemblyImportSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }

    return section
}

func (section *WebAssemblyImportSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder

    memoryCount := 0
    tableCount := 0
    globalCount := 0
    functionCount := 0

    for i, item := range section.Items {
        out.WriteString(indents)
        out.WriteString("(import ")

        switch item.Kind.(type) {
            case *FunctionImport:
                func_ := item.Kind.(*FunctionImport)
                out.WriteString(fmt.Sprintf("\"%v\" \"%v\" (func (;%v;) (type %v))", item.ModuleName, item.Name, functionCount, func_.Index))
                functionCount += 1
            case *GlobalType:
                global := item.Kind.(*GlobalType)
                out.WriteString(fmt.Sprintf("\"%v\" \"%v\" (global (;%v;) ", item.ModuleName, item.Name, globalCount))
                if global.Mutable {
                    out.WriteString(fmt.Sprintf("(mut %v)", global.ValueType.ConvertToWat(indents)))
                } else {
                    out.WriteString(global.ValueType.ConvertToWat(indents))
                }

                globalCount += 1
            case *MemoryImportType:
                memory := item.Kind.(*MemoryImportType)
                out.WriteString(fmt.Sprintf("\"%v\" \"%v\" (memory (;%v;) %v", item.ModuleName, item.Name, memoryCount, memory.Limit.Minimum))
                if memory.Limit.HasMaximum {
                    out.WriteString(fmt.Sprintf(" %v", memory.Limit.Maximum))
                }
                out.WriteByte(')')
                memoryCount += 1
            case *TableType:
                table := item.Kind.(*TableType)
                out.WriteString(fmt.Sprintf("\"%v\" \"%v\" (table (;%v;) %v ", item.ModuleName, item.Name, tableCount, table.Limit.Minimum))
                if table.Limit.HasMaximum {
                    out.WriteString(fmt.Sprintf("%v ", table.Limit.Maximum))
                }

                switch table.RefType {
                    case RefTypeFunction:
                        out.WriteString("funcref")
                    case RefTypeExtern:
                        out.WriteString("externref")
                }
                out.WriteByte(')')
                tableCount += 1
            default:
                out.WriteString(fmt.Sprintf("unhandled import index=%v type %+v", i, item.Kind))
        }

        out.WriteByte(')')
        if i < len(section.Items) - 1 {
            out.WriteByte('\n')
        }
    }

    return out.String()
}

func (section *WebAssemblyImportSection) String() string {
    return "import section"
}

func (section *WebAssemblyImportSection) AddImport(moduleName string, name string, kind Index) {
    section.Items = append(section.Items, ImportSectionItem{
        ModuleName: moduleName,
        Name: name,
        Kind: kind,
    })
}

type WebAssemblyTypeSection struct {
    Functions []WebAssemblyFunction
}

func (section *WebAssemblyTypeSection) ToInterface() WebAssemblySection {
    if section == nil {
        return nil
    }
    return section
}

func (section *WebAssemblyTypeSection) GetFunction(index uint32) WebAssemblyFunction {
    if index < uint32(len(section.Functions)) {
        return section.Functions[index]
    }

    return WebAssemblyFunction{}
}

func (section *WebAssemblyTypeSection) AddFunctionType(function WebAssemblyFunction) {
    section.Functions = append(section.Functions, function)
}

/* adds the function type to the list of function types and returns its index, or
 * just returns the index of an existing type
 */
func (section *WebAssemblyTypeSection) GetOrCreateFunctionType(function WebAssemblyFunction) uint32 {
    for i, check := range section.Functions {
        if check.Equals(function) {
            return uint32(i)
        }
    }

    section.AddFunctionType(function)
    return uint32(len(section.Functions) - 1)
}

func (section *WebAssemblyTypeSection) ConvertToWat(module *WebAssemblyModule, indents string) string {
    var out strings.Builder
    for i, function := range section.Functions {
        out.WriteString(indents)
        out.WriteString("(type ")
        out.WriteString(fmt.Sprintf("(;%v;)", i))
        out.WriteString(" (func")

        if len(function.InputTypes) > 0 {
            out.WriteString(" (param")
            for _, input := range function.InputTypes {
                out.WriteByte(' ')
                out.WriteString(input.ConvertToWat(""))
            }
            out.WriteByte(')')
        }

        if len(function.OutputTypes) > 0 {
            out.WriteString(" (result")
            for _, output := range function.OutputTypes {
                out.WriteByte(' ')
                out.WriteString(output.ConvertToWat(""))
            }
            out.WriteString(")")
        }

        out.WriteByte(')') // for func
        out.WriteByte(')') // for type
        if i < len(section.Functions) {
            out.WriteByte('\n')
        }
    }
    return out.String()
}

func (section *WebAssemblyTypeSection) String() string {
    return "type section"
}

type WebAssemblyModule struct {
    Sections []WebAssemblySection
}

func (module *WebAssemblyModule) FindFunctionType(index int) *TypeIndex {
    for _, section := range module.Sections {
        function, ok := section.(*WebAssemblyFunctionSection)
        if ok {
            return function.GetFunctionType(index)
        }
    }

    return nil
}

func (module *WebAssemblyModule) GetFunction(index uint32) WebAssemblyFunction {
    for _, section := range module.Sections {
        type_, ok := section.(*WebAssemblyTypeSection)
        if ok {
            return type_.GetFunction(index)
        }
    }

    return WebAssemblyFunction{}
}

func (module *WebAssemblyModule) GetImportFunctionCount() int {
    for _, section := range module.Sections {
        import_, ok := section.(*WebAssemblyImportSection)
        if ok {
            return import_.CountFunctions()
        }
    }

    return 0
}

func (module *WebAssemblyModule) AddSection(section WebAssemblySection) {
    module.Sections = append(module.Sections, section)
}

func (module *WebAssemblyModule) ConvertToWat(indents string) string {
    var out strings.Builder

    out.WriteString("(module\n")
    for _, section := range module.Sections {
        out.WriteString(section.ConvertToWat(module, "  "))
        out.WriteString("\n")
    }
    out.WriteString(")")

    return out.String()
}
