package main

import (
    "log"
    "os"
    "bufio"
    "io"
    "fmt"
    "bytes"
    "encoding/binary"
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
