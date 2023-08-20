package parser

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	COMPRESSION      = 0x02
	LZMA_COMPRESSION = 0x18
)

func Decompress7BitCompression(buf []byte) string {
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

	return strings.TrimSuffix(string(result), "\x00")
}

func ParseLongText(buf []byte, flag uint32) string {
	if len(buf) < 2 {
		return ""
	}

	//fmt.Printf("Record Flag %v\n", flag)
	start := 0
	if flag != 1 {
		flag = uint32(buf[0])
		start++
	}
	//fmt.Printf("Inline Flag %v\n", flag)

	// Lzxpress compression - not supported right now.
	if flag&COMPRESSION != 0 {
		if flag == 0x18 {
			fmt.Printf("LZXPRESS compression not supported currently\n")
			return string(buf)
		}
		return Decompress7BitCompression(buf[start:])
	}
	return ParseTerminatedUTF16String(&BufferReaderAt{buf[start:]}, 0)
}
