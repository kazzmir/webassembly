package core

import (
    "log"
    "os"
    "bufio"
    "io"
    "fmt"
    "bytes"
    "errors"
    "encoding/binary"
    // "encoding/hex"
)

func isAsmBytes(asm []byte) bool {
    if len(asm) != 4 {
        return false
    }

    return bytes.Compare(asm, []byte{0, 'a', 's', 'm'}) == 0
}

type WebAssemblyFileModule struct {
    path string
    io io.ReadCloser
    reader *bufio.Reader
    debug bool
}

func WebAssemblyNew(path string, debug bool) (WebAssemblyFileModule, error) {
    file, err := os.Open(path)
    if err != nil {
        return WebAssemblyFileModule{}, err
    }

    return WebAssemblyFileModule{
        path: path,
        io: file,
        reader: bufio.NewReader(file),
        debug: debug,
    }, nil
}

func (module *WebAssemblyFileModule) ReadMagic() error {
    asmBytes := make([]byte, 4)
    count, err := io.ReadFull(module.reader, asmBytes)
    if count != len(asmBytes) {
        if err != nil {
            return err
        }

        return fmt.Errorf("Failed to read asm bytes from '%v'\n", module.path)
    }

    if !isAsmBytes(asmBytes){
        return fmt.Errorf("Unable to read the module preamble from '%v'. Not a webassembly file?", module.path)
    }

    return nil
}

func (module *WebAssemblyFileModule) ReadVersion() (uint32, error) {
    var version uint32
    err := binary.Read(module.reader, binary.LittleEndian, &version)
    if err != nil {
        return 0, err
    }

    return version, nil
}

func (module *WebAssemblyFileModule) Close() {
    module.io.Close()
}

func (module *WebAssemblyFileModule) ReadSectionId() (byte, error) {
    return module.reader.ReadByte()
}

/* All integers are encoded using the LEB128 variable-length integer encoding, in either unsigned or signed variant.
 * Unsigned integers are encoded in unsigned LEB128 format. As an additional constraint, the total number of bytes encoding a value of type uNuN must not exceed ceil(N/7)ceil(N/7) bytes.
 */
func (module *WebAssemblyFileModule) ReadU32() (uint32, error) {
    return ReadU32(module.reader)
}

type ValueType int
const (
    InvalidValueType ValueType = 0
    ValueTypeI32 ValueType = 0x7f
    ValueTypeI64 ValueType = 0x7e
    ValueTypeF32 ValueType = 0x7d
    ValueTypeF64 ValueType = 0x7c
)

func (value *ValueType) ConvertToWat(indents string) string {
    switch *value {
        case InvalidValueType: return "invalid"
        case ValueTypeI32: return "i32"
        case ValueTypeI64: return "i64"
        case ValueTypeF32: return "f32"
        case ValueTypeF64: return "f64"
        default: return "?"
    }
}

func ReadValueType(reader io.ByteReader) (ValueType, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return InvalidValueType, err
    }

    switch kind {
        case 0x7f: return ValueTypeI32, nil
        case 0x7e: return ValueTypeI64, nil
        case 0x7d: return ValueTypeF32, nil
        case 0x7c: return ValueTypeF64, nil
    }

    return InvalidValueType, fmt.Errorf("Unknown value type %v", kind)
}

func ReadTypeVector(reader io.ByteReader) ([]ValueType, error) {
    length, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    var out []ValueType
    var i uint32
    for i = 0; i < length; i++ {
        valueType, err := ReadValueType(reader)
        if err != nil {
            return nil, err
        }
        out = append(out, valueType)
    }

    return out, nil
}

const FunctionTypeMagic = 0x60

type WebAssemblyFunction struct {
    InputTypes []ValueType
    OutputTypes []ValueType
}

func (module *WebAssemblyFileModule) ReadFunctionType(reader io.Reader) (WebAssemblyFunction, error) {
    buffer := NewByteReader(reader)
    magic, err := buffer.ReadByte()
    if err != nil {
        return WebAssemblyFunction{}, fmt.Errorf("Could not read function type: %v", err)
    }

    if magic != FunctionTypeMagic {
        return WebAssemblyFunction{}, fmt.Errorf("Expected to read function type 0x%x but got 0x%x", magic, FunctionTypeMagic)
    }

    inputTypes, err := ReadTypeVector(buffer)
    if err != nil {
        return WebAssemblyFunction{}, err
    }

    outputTypes, err := ReadTypeVector(buffer)
    if err != nil {
        return WebAssemblyFunction{}, err
    }

    if module.debug {
        log.Printf("Function %v -> %v\n", inputTypes, outputTypes)
    }

    return WebAssemblyFunction{
        InputTypes: inputTypes,
        OutputTypes: outputTypes,
    }, nil
}

func (module *WebAssemblyFileModule) ReadTypeSection(sectionSize uint32) (*WebAssemblyTypeSection, error) {
    if module.debug {
        log.Printf("Type section size %v\n", sectionSize)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(sectionSize)))
    /*
    sectionReader := &io.LimitedReader{
        R: module.reader,
        N: int64(sectionSize),
    }
    */

    // log.Printf("Bytes remaining %v\n", sectionReader.N)

    length, err := ReadU32(sectionReader)
    if err != nil {
        return nil, err
    }

    // log.Printf("Read %v function types\n", length)

    var section WebAssemblyTypeSection

    var i uint32
    for i = 0; i < length; i++ {

        // log.Printf("Bytes remaining %v\n", sectionReader.N)

        function, err := module.ReadFunctionType(sectionReader)
        if err != nil {
            return &section, err
        }

        section.AddFunctionType(function)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading type section: not all bytes were read")
    }

    return &section, nil
}

type ImportDescription byte
const (
    InvalidImportDescription ImportDescription = 0xff
    FunctionImportDescription ImportDescription = 0x00
    TableImportDescription ImportDescription = 0x01
    MemoryImportDescription ImportDescription = 0x02
    GlobalImportDescription ImportDescription = 0x03
)

type Index interface {
    String() string
}

type FunctionIndex struct {
    Id uint32
}

func (index *FunctionIndex) String() string {
    return fmt.Sprintf("function index %v", index.Id)
}

func ReadFunctionIndex(reader *ByteReader) (*FunctionIndex, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    return &FunctionIndex{
        Id: index,
    }, nil
}

func ReadFunctionIndexVector(reader *ByteReader) ([]*FunctionIndex, error) {
    var out []*FunctionIndex
    length, err := ReadU32(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read function index vector: %v", err)
    }

    var j uint32
    for j = 0; j < length; j++ {
        functionIndex, err := ReadFunctionIndex(reader)
        if err != nil {
            return nil, fmt.Errorf("Could not read function index %v: %v", j, err)
        }

        out = append(out, functionIndex)
    }

    return out, nil
}

type TypeIndex struct {
    Index
    Id uint32
}

func (index *TypeIndex) String() string {
    return fmt.Sprintf("(type %v)", index.Id)
}

func ReadTypeIndex(reader *ByteReader) (*TypeIndex, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    return &TypeIndex{
        Id: index,
    }, nil
}

type TableIndex struct {
    Index
    Id uint32
}

func (index *TableIndex) String() string {
    return fmt.Sprintf("table index %v", index.Id)
}

func ReadTableIndex(reader *ByteReader) (*TableIndex, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    return &TableIndex{
        Id: index,
    }, nil
}

type GlobalType struct {
    Index
    ValueType ValueType
    Mutable bool
}

func (global *GlobalType) String() string {
    return fmt.Sprintf("global type value type=0x%x mutable=%v", global.ValueType, global.Mutable)
}

func ReadGlobalType(reader *ByteReader) (*GlobalType, error) {
    value, err := ReadValueType(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read value type for global index: %v", err)
    }

    mutable, err := reader.ReadByte()
    if err != nil {
        return nil, fmt.Errorf("Could not read mutable state for global index: %v", err)
    }

    if mutable != 0 && mutable != 1 {
        return nil, fmt.Errorf("Global index mutable value was not 0 (const) or 1 (var), instead it was %v", mutable)
    }

    isMutable := mutable == 1

    return &GlobalType{
        ValueType: value,
        Mutable: isMutable,
    }, nil
}

func ReadLocalIndex(reader *ByteReader) (uint32, error) {
    return ReadU32(reader)
}

type MemoryImportType struct {
    Import
    Limit Limit
}

func (memory *MemoryImportType) String() string {
    return fmt.Sprintf("memory %v", memory.Limit)
}

func ReadMemoryType(reader *ByteReader) (*MemoryImportType, error) {
    limit, err := ReadLimit(reader)
    if err != nil {
        return nil, err
    }

    return &MemoryImportType{Limit: limit}, nil
}

/* FIXME: not sure if this is needed */
type Import interface {
    String() string
}

type FunctionImport struct {
    Index uint32
}

func (index *FunctionImport) String() string {
    return fmt.Sprintf("function import index=%v", index.Index)
}

func ReadFunctionImport(reader *ByteReader) (*FunctionImport, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read function index: %v", err)
    }

    return &FunctionImport{Index: index}, nil
}

func ReadImportDescription(reader *ByteReader) (Index, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return nil, err
    }

    switch kind {
        case byte(FunctionImportDescription): return ReadFunctionImport(reader)
        case byte(TableImportDescription): return ReadTableType(reader)
        case byte(MemoryImportDescription): return ReadMemoryType(reader)
        case byte(GlobalImportDescription): return ReadGlobalType(reader)
    }

    return nil, fmt.Errorf("Unknown import description '%v'", kind)
}

func (module *WebAssemblyFileModule) ReadImportSection(sectionSize uint32) (*WebAssemblyImportSection, error) {
    if module.debug {
        log.Printf("Read import section size %v\n", sectionSize)
    }

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(sectionSize)))

    imports, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read imports: %v", err)
    }

    if module.debug {
        log.Printf("Have %v imports\n", imports)
    }

    var section WebAssemblyImportSection

    var i uint32
    for i = 0; i < imports; i++ {
        moduleName, err := ReadName(sectionReader)
        if err != nil {
            return nil, err
        }

        name, err := ReadName(sectionReader)
        if err != nil {
            return nil, err
        }

        kind, err := ReadImportDescription(sectionReader)
        if err != nil {
            return nil, err
        }

        if module.debug {
            log.Printf("Import %v: module='%v' name='%v' kind='%v'\n", i, moduleName, name, kind.String())
        }

        section.AddImport(moduleName, name, kind)
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadFunctionSection(size uint32) (*WebAssemblyFunctionSection, error) {
    if module.debug {
        log.Printf("Read function section size %v\n", size)
    }

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))
    types, err := ReadU32(sectionReader)
    if err != nil {
        return nil, err
    }

    var section WebAssemblyFunctionSection

    if module.debug {
        log.Printf("Functions %v\n", types)
    }

    var i uint32
    for i = 0; i < types; i++ {
        index, err := ReadTypeIndex(sectionReader)
        if err != nil {
            return nil, err
        }

        if module.debug {
            log.Printf("Function %v has type index 0x%x\n", i, index)
        }

        section.AddFunction(index)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading function section: not all bytes were read")
    }

    return &section, nil
}

type MemoryIndex struct {
    Index
    Id uint32
}

func (memory *MemoryIndex) String() string {
    return fmt.Sprintf("memory index id=%v", memory.Id)
}

func ReadMemoryIndex(reader *ByteReader) (*MemoryIndex, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read memory index: %v", err)
    }

    return &MemoryIndex{Id: index}, nil
}

type GlobalIndex struct {
    Index
    Id uint32
}

func (global *GlobalIndex) String() string {
    return fmt.Sprintf("global %v", global.Id)
}

func ReadGlobalIndex(reader *ByteReader) (*GlobalIndex, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read global index: %v", err)
    }

    return &GlobalIndex{Id: index}, nil
}

type ExportDescription byte
const (
    InvalidExportDescription ExportDescription = 0xff
    FunctionExportDescription ExportDescription = 0x00
    TableExportDescription ExportDescription = 0x01
    MemoryExportDescription ExportDescription = 0x02
    GlobalExportDescription ExportDescription = 0x03
)

func ReadExportDescription(reader *ByteReader) (Index, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return nil, err
    }

    switch kind {
        case byte(FunctionExportDescription): return ReadFunctionIndex(reader)
        case byte(TableExportDescription): return ReadTableIndex(reader)
        case byte(MemoryExportDescription): return ReadMemoryIndex(reader)
        case byte(GlobalExportDescription): return ReadGlobalIndex(reader)
    }

    return nil, fmt.Errorf("Unknown import description '%v'", kind)
}

func (module *WebAssemblyFileModule) ReadExportSection(size uint32) (*WebAssemblyExportSection, error) {
    if module.debug {
        log.Printf("Read export section size %v\n", size)
    }

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    exports, err := ReadU32(sectionReader)
    if err != nil {
        return nil, err
    }

    var section WebAssemblyExportSection

    var i uint32
    for i = 0; i < exports; i++ {
        name, err := ReadName(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read name from export section for export %v: %v", i, err)
        }

        description, err := ReadExportDescription(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read description for export %v: %v", i, err)
        }

        if module.debug {
            log.Printf("Export %v: name='%v' description=%v\n", i, name, description)
        }

        section.AddExport(name, description)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading export section: not all bytes were read")
    }

    return &section, nil
}

type MemoryArgument struct {
    Align uint32
    Offset uint32
}

func ReadMemoryArgument(reader *ByteReader) (MemoryArgument, error) {
    align, err := ReadU32(reader)
    if err != nil {
        return MemoryArgument{}, fmt.Errorf("Could not read alignment of memory argument: %v", err)
    }

    offset, err := ReadU32(reader)
    if err != nil {
        return MemoryArgument{}, fmt.Errorf("Could not read offset of memory argument: %v", err)
    }

    return MemoryArgument{
        Align: align,
        Offset: offset,
    }, nil
}

func (module *WebAssemblyFileModule) ReadCodeSection(size uint32) (*WebAssemblyCodeSection, error) {
    if module.debug {
        log.Printf("Read code section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    codes, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read code vector length from code section: %v", err)
    }

    var section WebAssemblyCodeSection

    var i uint32
    for i = 0; i < codes; i++ {
        size, err := ReadU32(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Error reading size of code %v: %v", i, err)
        }

        if module.debug {
            log.Printf("Reading code entry %v size %v\n", i, size)
        }

        codeReader := NewByteReader(io.LimitReader(sectionReader, int64(size)))
        code, err := ReadCode(codeReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read code %v: %v", i, err)
        }

        _, err = codeReader.ReadByte()
        if err == nil {
            return nil, fmt.Errorf("Error reading code %v: not all bytes were read", i)
        }

        if module.debug {
            log.Printf("Code %v: size=%v code=%v\n", i, size, code)
        }

        section.AddCode(code)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading code section: not all bytes were read")
    }

    return &section, nil
}

type Limit struct {
    Minimum uint32
    Maximum uint32
    HasMaximum bool
}

func ReadLimit(reader *ByteReader) (Limit, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return Limit{}, fmt.Errorf("Could not read limit type: %v", err)
    }

    switch kind {
        case 0x00:
            minimum, err := ReadU32(reader)
            if err != nil {
                return Limit{}, fmt.Errorf("Could not read minimum limit: %v", err)
            }
            return Limit{
                Minimum: minimum,
                HasMaximum: false,
            }, nil
        case 0x01:
            minimum, err := ReadU32(reader)
            if err != nil {
                return Limit{}, fmt.Errorf("Could not read minimum limit: %v", err)
            }
            maximum, err := ReadU32(reader)
            if err != nil {
                return Limit{}, fmt.Errorf("Could not read maximum limit: %v", err)
            }

            return Limit{
                Minimum: minimum,
                Maximum: maximum,
                HasMaximum: true,
            }, nil
    }

    return Limit{}, fmt.Errorf("Unknown limit type 0x%x", kind)
}

const RefTypeFunction = 0x70
const RefTypeExtern = 0x69

func (module *WebAssemblyFileModule) ReadTableSection(size uint32) (*WebAssemblyTableSection, error) {
    if module.debug {
        log.Printf("Read table section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    tables, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read number of tables in the table section: %v", err)
    }

    var section WebAssemblyTableSection

    var i uint32
    for i = 0; i < tables; i++ {

        tableType, err := ReadTableType(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read table type in table section %v: %v", i, err)
        }
        section.AddTable(*tableType)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading table section: not all bytes were read")
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadElementSection(size uint32) (*WebAssemblyElementSection, error) {
    if module.debug {
        log.Printf("Read element section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    elements, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read elements length from the element section: %v", err)
    }

    var section WebAssemblyElementSection

    var i uint32
    for i = 0; i < elements; i++ {
        index, err := ReadU32(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read type index for element %v: %v", i, err)
        }

        switch index {
            /* 0:u32 e:expr y*:vec(funcidx) ->
             * {type funcref, init ((ref.func y) end)*, mode active {table 0, offset e}}
             */
            case 0:
                expressions, _, err := ReadExpressionSequence(sectionReader, false)
                if err != nil {
                    return nil, fmt.Errorf("Could not read expressions for element %v: %v", i, err)
                }

                functions, err := ReadFunctionIndexVector(sectionReader)
                if err != nil {
                    return nil, fmt.Errorf("Could not read function index vector for element %v: %v", i, err)
                }

                if module.debug {
                    log.Printf("Element %v: index=%v expressions=%v\n", i, index, expressions)
                }

                section.AddFunctionRefInit(functions, expressions)
            case 1:
            case 2:
            case 3:
            case 4:
            case 5:
            case 6:
            case 7:
        }
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading element section: not all bytes were read")
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadCustomSection(size uint32) (*WebAssemblyCustomSection, error) {
    if module.debug {
        log.Printf("Read custom section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    name, err := ReadName(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read name from custom section: %v", err)
    }

    var section WebAssemblyCustomSection

    for {
        raw, err := sectionReader.ReadByte()
        if errors.Is(err, io.EOF){
            break
        }
        if err != nil {
            return nil, fmt.Errorf("Could not read bytes from custom section: %v", err)
        }

        _ = raw
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading custom section: not all bytes were read")
    }

    if module.debug {
        log.Printf("Custom section '%v'\n", name)
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadMemorySection(size uint32) (*WebAssemblyMemorySection, error) {
    if module.debug {
        log.Printf("Read memory section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    memories, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read number of memory elements in memory section: %v", memories)
    }

    var section WebAssemblyMemorySection

    var i uint32
    for i = 0; i < memories; i++ {
        limit, err := ReadLimit(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read memory element %v: %v", i, err)
        }

        if module.debug {
            log.Printf("Read memory element %v: %v\n", i, limit)
        }

    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadGlobalSection(size uint32) (*WebAssemblyGlobalSection, error) {
    if module.debug {
        log.Printf("Read global section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    globals, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read number of entries in the global section: %v", err)
    }

    var section WebAssemblyGlobalSection

    var i uint32
    for i = 0; i < globals; i++ {
        global, err := ReadGlobalType(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read global element %v: %v", i, err)
        }

        expressions, _, err := ReadExpressionSequence(sectionReader, false)
        if err != nil {
            return nil, fmt.Errorf("Could not read expressions for global %v: %v", i, err)
        }

        if module.debug {
            log.Printf("Global element %v: global=%v expressions=%v\n", i, global, expressions)
        }

        section.AddGlobal(global, expressions)
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadDataSection(size uint32) (*WebAssemblyDataSection, error) {
    if module.debug {
        log.Printf("Read data section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    datas, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read number of entries in the data section: %v", err)
    }

    var section WebAssemblyDataSection

    var i uint32
    for i = 0; i < datas; i++ {
        kind, err := ReadU32(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read memory index for data entry %v: %v", i, err)
        }

        switch kind {
            /* active memory=0, offset=e */
            case 0:
                expressions, _, err := ReadExpressionSequence(sectionReader, false)
                if err != nil {
                    return nil, fmt.Errorf("Could not read expressions for data entry %v: %v", i, err)
                }

                data, err := ReadByteVector(sectionReader)
                if err != nil {
                    return nil, fmt.Errorf("Could not read init vector for data entry %v: %v", i, err)
                }

                if module.debug {
                    log.Printf("Data entry %v: kind=%v expressions=%v init-vector=%v\n", i, kind, expressions, data)
                }

                section.AddData(data, &MemoryActiveMode{
                    Memory: 0,
                    Offset: expressions,
                })

            /* passive */
            case 1:
                data, err := ReadByteVector(sectionReader)
                if err != nil {
                    return nil, fmt.Errorf("Could not read init vector for data entry %v: %v", i, err)
                }
                section.AddData(data, &MemoryPassiveMode{})

            /* active memory=x, offset=e */
            case 2:
                memoryIndex, err := ReadMemoryIndex(sectionReader)
                if err != nil {
                    return nil, fmt.Errorf("Could not read memory index for data entry %v: %v", i, err)
                }

                expressions, _, err := ReadExpressionSequence(sectionReader, false)
                if err != nil {
                    return nil, fmt.Errorf("Could not read expressions for data entry %v: %v", i, err)
                }

                data, err := ReadByteVector(sectionReader)
                if err != nil {
                    return nil, fmt.Errorf("Could not read init vector for data entry %v: %v", i, err)
                }

                section.AddData(data, &MemoryActiveMode{
                    Memory: memoryIndex.Id,
                    Offset: expressions,
                })

            default:
                return nil, fmt.Errorf("invalid data component with value %v", kind)

        }
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading data section: not all bytes were read")
    }

    return &section, nil
}

func (module *WebAssemblyFileModule) ReadStartSection(size uint32) (*WebAssemblyStartSection, error) {
    if module.debug {
        log.Printf("Read start section size %v\n", size)
    }
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    startIndex, err := ReadFunctionIndex(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read start function index: %v", err)
    }

    return &WebAssemblyStartSection{
        Start: *startIndex,
    }, nil
}

const (
    CustomSection byte = 0
    TypeSection byte = 1
    ImportSection byte = 2
    FunctionSection byte = 3
    TableSection byte = 4
    MemorySection byte = 5
    GlobalSection byte = 6
    ExportSection byte = 7
    StartSection byte = 8
    ElementSection byte = 9
    CodeSection byte = 10
    DataSection byte = 11
)

func (module *WebAssemblyFileModule) ReadSection() (WebAssemblySection, error) {
    sectionId, err := module.ReadSectionId()
    if err != nil {
        /* If we read eof then we probably read all bytes available, so there is no section to read */
        if errors.Is(err, io.EOF) {
            return nil, nil
        }

        return nil, fmt.Errorf("Could not read section id: %v", err)
    }

    sectionSize, err := module.ReadU32()
    if err != nil {
        return nil, fmt.Errorf("Could not read section size: %v", err)
    }

    switch sectionId {
        case CustomSection:
            out, err := module.ReadCustomSection(sectionSize)
            return out.ToInterface(), err
        case TypeSection:
            out, err := module.ReadTypeSection(sectionSize)
            return out.ToInterface(), err
        case ImportSection:
            out, err := module.ReadImportSection(sectionSize)
            return out.ToInterface(), err
        case FunctionSection:
            out, err := module.ReadFunctionSection(sectionSize)
            return out.ToInterface(), err
        case TableSection:
            out, err := module.ReadTableSection(sectionSize)
            return out.ToInterface(), err
        case MemorySection:
            out, err := module.ReadMemorySection(sectionSize)
            return out.ToInterface(), err
        case GlobalSection:
            out, err := module.ReadGlobalSection(sectionSize)
            return out.ToInterface(), err
        case ExportSection:
            out, err := module.ReadExportSection(sectionSize)
            return out.ToInterface(), err
        case StartSection:
            out, err := module.ReadStartSection(sectionSize)
            return out.ToInterface(), err
        case ElementSection:
            out, err := module.ReadElementSection(sectionSize)
            return out.ToInterface(), err
        case CodeSection:
            out, err := module.ReadCodeSection(sectionSize)
            return out.ToInterface(), err
        case DataSection:
            out, err := module.ReadDataSection(sectionSize)
            return out.ToInterface(), err
    }

    return nil, fmt.Errorf("Unknown section id %v", sectionId)
}

/* Contains all the sections */
/*
type WebAssemblyModule struct {
    TypeSection *WebAssemblyTypeSection
    ImportSection *WebAssemblyImportSection
    FunctionSection *WebAssemblyFunctionSection
    TableSection *WebAssemblyTableSection
    ExportSection *WebAssemblyExportSection
    ElementSection *WebAssemblyElementSection
    CodeSection *WebAssemblyCodeSection
}
*/

func ParseWasmFile(path string, debug bool) (WebAssemblyModule, error) {
    module, err := WebAssemblyNew(path, debug)
    if err != nil {
        return WebAssemblyModule{}, err
    }

    defer module.Close()

    err = module.ReadMagic()
    if err != nil {
        return WebAssemblyModule{}, err
    }

    version, err := module.ReadVersion()
    if err != nil {
        return WebAssemblyModule{}, err
    }

    if version != 1 {
        log.Printf("Warning: unexpected module version %v. Expected 1\n", version)
    }

    var moduleOut WebAssemblyModule

    for {
        section, err := module.ReadSection()
        if err != nil {
            return WebAssemblyModule{}, err
        }

        if section == nil {
            return moduleOut, nil
        }

        if module.debug {
            log.Printf("Read section '%v'\n", section.String())
        }
        moduleOut.AddSection(section)
    }

    return WebAssemblyModule{}, nil
}
