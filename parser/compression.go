package parser

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

func Decompress7BitCompression(buf []byte) []byte {
	result := make([]byte, 0, (len(buf)+5)*8/7)

	value_16bit := uint16(0)
	bit_index := 0

	for i := 1; i < len(buf); i++ {
		slice := buf[i : i+1]
		if i+1 < len(buf) {
			slice = append(slice, buf[i+1])
		}

		for len(slice) < 2 {
			slice = append(slice, 0)
		}

		value_16bit |= binary.LittleEndian.Uint16(slice) << bit_index
		result = append(result, byte(value_16bit&0x7f))

		value_16bit >>= 7
		bit_index++

		if bit_index == 7 {
			result = append(result, byte(value_16bit&0x7f))
			value_16bit >>= 7
			bit_index = 0
		}
	}
	return result
}

func DecompressLongValue(buf []byte) []byte {
	compression_flag := buf[0] >> 3
	switch {
	case compression_flag == 0x1:
		return Decompress7BitCompression(buf)
	case compression_flag == 0x2:
		decompressed := Decompress7BitCompression(buf)
		decompressedUTF16 := make([]byte, len(decompressed)*2)
		// Technically not needed but simplifies the calling code since the column codepage is Unicode
		for i := range decompressed {
			decompressedUTF16[2*i] = decompressed[i]
		}
		return decompressedUTF16
	case compression_flag == 0x3:
		fmt.Printf("LZXPRESS compression not supported currently\n")
		return nil
	default:
		fmt.Printf("Unknown compression flag: %d\n", compression_flag)
		return nil
	}
}

func ParseLongText(buf []byte, cp uint32) string {
	//cp == 0 is interpreted as ASCII (see upstream)
	if cp == 0 || cp == 1252 {
		return strings.Split(string(buf), "\x00")[0]
	} else if cp == 1200 {
		new_buf := UTF16BytesToUTF8(buf, binary.LittleEndian)
		return strings.Split(new_buf, "\x00")[0]
	} else {
		if Debug {
			fmt.Printf("Unexpected code page: %d for value %x\n", cp, buf)
		}
		return ""
	}
}

func ParseText(reader io.ReaderAt, offset int64, len int64, flags uint32) string {
	if len < 0 {
		return ""

	}
	if len > 1024*10 {
		len = 1024 * 10
	}

	data := make([]byte, len)
	n, err := reader.ReadAt(data, offset)
	if err != nil {
		return ""
	}
	data = data[:n]

	var str string
	if flags == 1 {
		str = string(data[:n])
	} else {
		str = UTF16BytesToUTF8(data, binary.LittleEndian)
	}
	return strings.Split(str, "\x00")[0]
}
