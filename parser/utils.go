package parser

import (
	"encoding/binary"
	"io"
	"time"
)

func WinFileTime64(reader io.ReaderAt, offset int64) time.Time {
	value := ParseInt64(reader, offset)
	return time.Unix((value/10000000)-11644473600, 0).UTC()
}

func WinFileTime64Bin(bytes []byte) time.Time {
	value := int64(binary.LittleEndian.Uint64(bytes))
	return time.Unix((value/10000000)-11644473600, 0).UTC()
}

func IsSmallPage(page_size int64) bool {
	return page_size <= 1024*8
}
