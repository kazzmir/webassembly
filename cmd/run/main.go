package main

import (
    "log"
    "os"
    "bufio"
    "io"
    "fmt"
    "bytes"
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

func (module *WebAssemblyFileModule) ReadFunctionType(reader io.Reader) error {
    buffer := NewByteReader(reader)
    magic, err := buffer.ReadByte()
    if err != nil {
        return fmt.Errorf("Could not read function type: %v", err)
    }

    if magic != 0x60 {
        return fmt.Errorf("Expected to read function type 0x60 but got 0x%x\n", magic)
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
    // sectionReader := io.LimitReader(module.reader, int64(sectionSize))
    sectionReader := &io.LimitedReader{
        R: module.reader,
        N: int64(sectionSize),
    }

    log.Printf("Bytes remaining %v\n", sectionReader.N)

    length, err := ReadU32(NewByteReader(sectionReader))
    if err != nil {
        return nil, err
    }

    log.Printf("Read %v function types\n", length)

    for length > 0 {
        length -= 1

        log.Printf("Bytes remaining %v\n", sectionReader.N)

        err := module.ReadFunctionType(sectionReader)
        if err != nil {
            return nil, err
        }
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

type ImportType interface {
}

type FunctionImport struct {
    ImportType
    Index uint32
}

func ReadFunctionImport(reader *ByteReader) (*FunctionImport, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    return &FunctionImport{
        Index: index,
    }, nil
}

type TableImport struct {
    ImportType
    Index uint32
}

func ReadTableImport(reader *ByteReader) (*TableImport, error) {
    index, err := ReadU32(reader)
    if err != nil {
        return nil, err
    }

    return &TableImport{
        Index: index,
    }, nil
}

func ReadImportDescription(reader *ByteReader) (ImportType, error) {
    kind, err := reader.ReadByte()
    if err != nil {
        return InvalidImportDescription, err
    }

    switch kind {
        case byte(FunctionImportDescription): return ReadFunctionImport(reader)
        case byte(TableImportDescription): return ReadTableImport(reader)
        case byte(MemoryImportDescription): return nil, fmt.Errorf("Unimplemented import description %v", MemoryImportDescription)
        case byte(GlobalImportDescription): return nil, fmt.Errorf("Unimplemented import description %v", GlobalImportDescription)
    }

    return InvalidImportDescription, fmt.Errorf("Unknown import description '%v'", kind)
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

        log.Printf("Import module '%v' name '%v' kind '%v'\n", moduleName, name, kind)
    }

    return nil, nil
}

func (module *WebAssemblyFileModule) ReadSection() (WebAssemblySection, error) {
    sectionId, err := module.ReadSectionId()
    if err != nil {
        return nil, err
    }

    sectionSize, err := module.ReadU32()
    if err != nil {
        return nil, err
    }

    switch sectionId {
        /* custom section */
        case 0: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* type section */
        case 1: return module.ReadTypeSection(sectionSize)
        /* import section */
        case 2: return module.ReadImportSection(sectionSize)
        /* function section */
        case 3: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* table section */
        case 4: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* memory section */
        case 5: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* global section */
        case 6: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* export section */
        case 7: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* start section */
        case 8: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* element section */
        case 9: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* code section */
        case 10: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
        /* data section */
        case 11: return nil, fmt.Errorf("Unimplemented section %v", sectionId)
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
