package core

import (
    "io"
    "fmt"
    "unicode/utf8"
    "encoding/binary"
)

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

func ReadU32(reader io.ByteReader) (uint32, error) {
    var result uint32
    var shift uint32

    count := 0

    var low byte = 0b1111111
    var high byte = 1 << 7

    for {
        next, err := reader.ReadByte()
        if err != nil {
            return 0, err
        }

        use := uint32(next & low)

        result = result | (use << shift)
        if next & high == 0 {
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

func ReadSignedLEB128(reader io.ByteReader, size int64) (int64, error) {
    var result int64
    var shift int64

    count := 0

    var low byte = 0b1111111
    var high byte = 1 << 7

    for {
        next, err := reader.ReadByte()
        if err != nil {
            return 0, err
        }

        use := int64(next & low)

        result = result | (use << shift)
        shift += 7

        if next & high == 0 {
            if shift < size && next & 0x40 == 0x40 {
                result = -result
            }

            return result, nil
        }

        /* Safety check */
        count += 1
        if count > 20 {
            return 0, fmt.Errorf("Read too many bytes in a LEB128 integer")
        }
    }

}

func ReadS32(reader io.ByteReader) (int32, error) {
    out, err := ReadSignedLEB128(reader, 32)
    return int32(out), err
}

func ReadS64(reader io.ByteReader) (int64, error) {
    return ReadSignedLEB128(reader, 64)
}

func ReadFloat32(reader io.Reader) (float32, error) {
    var value float32
    err := binary.Read(reader, binary.LittleEndian, &value)
    if err != nil {
        return 0, err
    }
    return value, nil
}

func ReadFloat64(reader io.Reader) (float64, error) {
    var value float64
    err := binary.Read(reader, binary.LittleEndian, &value)
    if err != nil {
        return 0, err
    }
    return value, nil
}

func ReadByteVector(reader *ByteReader) ([]byte, error) {
    bytes, err := ReadU32(reader)
    if err != nil {
        return nil, fmt.Errorf("Could not read vector size: %v", err)
    }

    vector := make([]byte, bytes)
    _, err = io.ReadFull(reader, vector)
    if err != nil {
        return nil, fmt.Errorf("Could not read %v bytes of vector: %v", bytes, err)
    }

    return vector, nil
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
