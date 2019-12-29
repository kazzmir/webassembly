package main

import (
    "log"
    "os"
    "bufio"
    "io"
    "fmt"
    "bytes"
    "errors"
    "unicode/utf8"
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
}

func WebAssemblyNew(path string) (WebAssemblyFileModule, error) {
    file, err := os.Open(path)
    if err != nil {
        return WebAssemblyFileModule{}, err
    }

    return WebAssemblyFileModule{
        path: path,
        io: file,
        reader: bufio.NewReader(file),
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

type WebAssemblySection interface {
    String() string
}

type WebAssemblyTypeSection struct {
    WebAssemblySection
}

func (section *WebAssemblyTypeSection) String() string {
    return "type section"
}

func ReadU32(reader io.ByteReader) (uint32, error) {
    var result uint32
    var shift uint32

    count := 0

    for {
        next, err := reader.ReadByte()
        if err != nil {
            return 0, err
        }

        result = result | uint32((next & 0b1111111) << shift)
        if next & (1<<7) == 0 {
            return result, nil
        }

        shift += 7

        /* Safety check */
        count += 1
        if count > 20 {
            return 0, fmt.Errorf("Read too many bytes in a LEB128 integer")
        }
    }
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

func (module *WebAssemblyFileModule) ReadFunctionType(reader io.Reader) error {
    buffer := NewByteReader(reader)
    magic, err := buffer.ReadByte()
    if err != nil {
        return fmt.Errorf("Could not read function type: %v", err)
    }

    if magic != FunctionTypeMagic {
        return fmt.Errorf("Expected to read function type 0x%x but got 0x%x\n", magic, FunctionTypeMagic)
    }

    inputTypes, err := ReadTypeVector(buffer)
    if err != nil {
        return err
    }

    outputTypes, err := ReadTypeVector(buffer)
    if err != nil {
        return err
    }

    log.Printf("Function %v -> %v\n", inputTypes, outputTypes)

    return nil
}

type ByteReader struct {
    io.ByteReader
    Reader io.Reader
}

func (reader *ByteReader) Read(data []byte) (int, error) {
    return reader.Reader.Read(data)
}

func (reader *ByteReader) ReadByte() (byte, error) {
    out := make([]byte, 1)
    count, err := reader.Reader.Read(out)
    if err != nil {
        return 0, err
    }

    if count == 1 {
        return out[0], nil
    }

    return 0, fmt.Errorf("Did not read a byte")
}

func NewByteReader(reader io.Reader) *ByteReader {
    return &ByteReader{
        Reader: reader,
    }
}

func (module *WebAssemblyFileModule) ReadTypeSection(sectionSize uint32) (*WebAssemblyTypeSection, error) {
    log.Printf("Type section size %v\n", sectionSize)
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

    var i uint32
    for i = 0; i < length; i++ {

        // log.Printf("Bytes remaining %v\n", sectionReader.N)

        err := module.ReadFunctionType(sectionReader)
        if err != nil {
            return nil, err
        }
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading type section: not all bytes were read")
    }


    return nil, nil
}

type WebAssemblyImportSection struct {
    WebAssemblySection
}

func (section *WebAssemblyImportSection) String() string {
    return "import section"
}

func ReadName(reader *ByteReader) (string, error) {
    length, err := ReadU32(reader)
    if err != nil {
        return "", fmt.Errorf("Could not read name length: %v", err)
    }

    /* FIXME: put limits somewhere */
    if length > 10 * 1024 * 1024 {
        return "", fmt.Errorf("Name length too large: %v", length)
    }

    raw := make([]byte, length)
    count, err := io.ReadFull(reader, raw)
    if err != nil {
        return "", fmt.Errorf("Could not read name bytes %v: %v", length, err)
    }

    if count != int(length) {
        return "", fmt.Errorf("Read %v out of %v bytes", count, length)
    }

    out := ""

    for len(raw) > 0 {
        next, size := utf8.DecodeRune(raw)
        if size == 0 {
            return "", fmt.Errorf("Could not decode utf8 string %v", raw)
        }

        out = out + string(next)

        raw = raw[size:]
    }

    return out, nil
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
}

type FunctionIndex struct {
    Index
    Id uint32
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

type TableIndex struct {
    Index
    Id uint32
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

func ReadImportDescription(reader *ByteReader) (Index, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return nil, err
    }

    switch kind {
        case byte(FunctionImportDescription): return ReadFunctionIndex(reader)
        case byte(TableImportDescription): return ReadTableIndex(reader)
        case byte(MemoryImportDescription): return nil, fmt.Errorf("Unimplemented import description %v", MemoryImportDescription)
        case byte(GlobalImportDescription): return nil, fmt.Errorf("Unimplemented import description %v", GlobalImportDescription)
    }

    return nil, fmt.Errorf("Unknown import description '%v'", kind)
}

func (module *WebAssemblyFileModule) ReadImportSection(sectionSize uint32) (*WebAssemblyImportSection, error) {
    log.Printf("Read import section size %v\n", sectionSize)

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(sectionSize)))

    imports, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read imports: %v", err)
    }

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

        log.Printf("Import %v: module='%v' name='%v' kind= '%v'\n", i, moduleName, name, kind)
    }

    return nil, nil
}

type WebAssemblyFunctionSection struct {
    WebAssemblySection
}

func (section *WebAssemblyFunctionSection) String() string {
    return "function section"
}

func (module *WebAssemblyFileModule) ReadFunctionSection(size uint32) (*WebAssemblyFunctionSection, error) {
    log.Printf("Read function section size %v\n", size)

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))
    types, err := ReadU32(sectionReader)
    if err != nil {
        return nil, err
    }

    log.Printf("Functions %v\n", types)

    var i uint32
    for i = 0; i < types; i++ {
        index, err := ReadU32(sectionReader)
        if err != nil {
            return nil, err
        }

        log.Printf("Function %v has type index 0x%x\n", i, index)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading function section: not all bytes were read")
    }


    return nil, nil
}

type WebAssemblyExportSection struct {
    WebAssemblySection
}

func (section *WebAssemblyExportSection) String() string {
    return "export section"
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
        case byte(MemoryExportDescription): return nil, fmt.Errorf("Unimplemented import description %v", MemoryExportDescription)
        case byte(GlobalExportDescription): return nil, fmt.Errorf("Unimplemented import description %v", GlobalExportDescription)
    }

    return nil, fmt.Errorf("Unknown import description '%v'", kind)
}

func (module *WebAssemblyFileModule) ReadExportSection(size uint32) (*WebAssemblyExportSection, error) {
    log.Printf("Read export section size %v\n", size)

    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    exports, err := ReadU32(sectionReader)
    if err != nil {
        return nil, err
    }

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

        log.Printf("Export %v: name='%v' description=%v\n", i, name, description)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading export section: not all bytes were read")
    }

    return nil, nil
}

type WebAssemblyCodeSection struct {
    WebAssemblySection
}

func (section *WebAssemblyCodeSection) String() string {
    return "code section"
}

type Code struct {
}

type Expression struct {
}

const (
    InstructionEnd = 0x0b
)

func ReadExpressionSequence(reader *ByteReader) ([]Expression, error){

    count := 0
    for {
        instruction, err := reader.ReadByte()
        if err != nil {
            return nil, fmt.Errorf("Could not read instruction: %v", err)
        }

        log.Printf("Instruction %v: 0x%x\n", count, instruction)

        count += 1

        if instruction == InstructionEnd {
            log.Printf("Read %v instructions\n", count)
            return nil, nil
        }
    }
}

func ReadCode(reader *ByteReader) (Code, error) {
    locals, err := ReadU32(reader)
    if err != nil {
        return Code{}, fmt.Errorf("Could not read locals: %v", err)
    }

    log.Printf("Read code locals %v\n", locals)

    var i uint32
    for i = 0; i < locals; i++ {
        size, err := ReadU32(reader)
        if err != nil {
            return Code{}, fmt.Errorf("Could not read local size for %v: %v", i, err)
        }

        valueType, err := ReadValueType(reader)
        if err != nil {
            return Code{}, fmt.Errorf("Could not read type of local for %v: %v", i, err)
        }

        log.Printf("Local %v; size=%v type=%v\n", i, size, valueType)
    }

    expressions, err := ReadExpressionSequence(reader)
    if err != nil {
        return Code{}, fmt.Errorf("Could not read expressions: %v", err)
    }

    _ = expressions

    return Code{}, nil
}

func (module *WebAssemblyFileModule) ReadCodeSection(size uint32) (*WebAssemblyCodeSection, error) {
    log.Printf("Read code section size %v\n", size)
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    codes, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read code vector length from code section: %v", err)
    }

    var i uint32
    for i = 0; i < codes; i++ {
        size, err := ReadU32(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Error reading size of code %v: %v", i, err)
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

        log.Printf("Code %v: size=%v code=%v\n", i, size, code)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading code section: not all bytes were read")
    }

    return nil, nil
}

type WebAssemblyTableSection struct {
    WebAssemblySection
}

func (section *WebAssemblyTableSection) String() string {
    return "table section"
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

const TableElementMagic = 0x70

func (module *WebAssemblyFileModule) ReadTableSection(size uint32) (*WebAssemblyTableSection, error) {
    log.Printf("Read table section size %v\n", size)
    sectionReader := NewByteReader(io.LimitReader(module.reader, int64(size)))

    tables, err := ReadU32(sectionReader)
    if err != nil {
        return nil, fmt.Errorf("Could not read number of tables in the table section: %v", err)
    }

    var i uint32
    for i = 0; i < tables; i++ {
        magic, err := sectionReader.ReadByte()
        if err != nil {
            return nil, fmt.Errorf("Could not read table entry %v: %v", i, err)
        }

        if magic != TableElementMagic {
            return nil, fmt.Errorf("Expected to read table magic 0x%x but got 0x%x for entry %v", TableElementMagic, magic, i)
        }

        limit, err := ReadLimit(sectionReader)
        if err != nil {
            return nil, fmt.Errorf("Could not read limit for table entry %v: %v", i, err)
        }

        log.Printf("Table entry %v: limit=%v\n", i, limit)
    }

    _, err = sectionReader.ReadByte()
    if err == nil {
        return nil, fmt.Errorf("Error reading table section: not all bytes were read")
    }

    return nil, nil
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
        case CustomSection: return nil, fmt.Errorf("Unimplemented section %v: custom section", sectionId)
        case TypeSection: return module.ReadTypeSection(sectionSize)
        case ImportSection: return module.ReadImportSection(sectionSize)
        case FunctionSection: return module.ReadFunctionSection(sectionSize)
        case TableSection: return module.ReadTableSection(sectionSize)
        case MemorySection: return nil, fmt.Errorf("Unimplemented section %v: memory section", sectionId)
        case GlobalSection: return nil, fmt.Errorf("Unimplemented section %v: global section", sectionId)
        case ExportSection: return module.ReadExportSection(sectionSize)
        case StartSection: return nil, fmt.Errorf("Unimplemented section %v: start section", sectionId)
        case ElementSection: return nil, fmt.Errorf("Unimplemented section %v: element section", sectionId)
        case CodeSection: return module.ReadCodeSection(sectionSize)
        case DataSection: return nil, fmt.Errorf("Unimplemented section %v: data section", sectionId)
    }

    return nil, fmt.Errorf("Unknown section id %v", sectionId)
}

func run(path string) error {
    module, err := WebAssemblyNew(path)
    if err != nil {
        return err
    }

    defer module.Close()

    err = module.ReadMagic()
    if err != nil {
        return err
    }

    version, err := module.ReadVersion()
    if err != nil {
        return err
    }

    if version != 1 {
        log.Printf("Warning: unexpected module version %v. Expected 1\n", version)
    }

    for {
        section, err := module.ReadSection()
        if err != nil {
            return err
        }
        if section == nil {
            return nil
        }

        log.Printf("Section %v\n", section.String())
    }

    return nil
}

func main(){
    log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
    log.Printf("Web assembly runner\n")

    if len(os.Args) > 1 {
        err := run(os.Args[1])
        if err != nil {
            log.Printf("Error: %v\n", err)
        }
    } else {
        log.Printf("Give a webassembly file to run\n")
    }
}
