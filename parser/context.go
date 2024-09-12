package parser

import (
	"errors"
	"fmt"
	"io"
)

type ESEContext struct {
	Reader   io.ReaderAt
	Profile  *ESEProfile
	PageSize int64
	Header   *FileHeader
	Version  uint32
	Revision uint32
}

func NewESEContext(reader io.ReaderAt) (*ESEContext, error) {
	result := &ESEContext{
		Profile: NewESEProfile(),
		Reader:  reader,
	}

	result.Header = result.Profile.FileHeader(reader, 0)
	if result.Header.Magic() != 0x89abcdef {
		return nil, errors.New(fmt.Sprintf(
			"Unsupported ESE file: Magic is %x should be 0x89abcdef",
			result.Header.Magic()))
	}

	result.PageSize = int64(result.Header.PageSize())
	switch result.PageSize {
	case 0x1000, 0x2000, 0x4000, 0x8000:
	default:
		return nil, errors.New(fmt.Sprintf(
			"Unsupported page size %x", result.PageSize))
	}

	result.Version = result.Header.FormatVersion()
	result.Revision = result.Header.FormatRevision()
	return result, nil
}

func (self *ESEContext) GetPage(id int64) *PageHeader {
	// First file page is file header, second page is backup of file
	// header.
	return &PageHeader{
		PageHeader_: self.Profile.PageHeader_(
			self.Reader, (id+1)*self.PageSize),
	}
}

func (self *ESEContext) IsSmallPage() bool {
	return self.PageSize <= 8192
}

func (self *ESEContext) MaskIb() uint16 {
	var offsetMask uint16
	if self.IsExtendedPageRevision() && !self.IsSmallPage() {
		offsetMask = 0x7fff
	} else {
		offsetMask = 0x1fff
	}
	return offsetMask
}

func (self *ESEContext) IsExtendedPageRevision() bool {
	return self.Revision >= 0x11
}

func (self *ESEContext) GetTaggedValueOffset(tagData uint16) uint16 {
	return tagData & self.MaskIb()
}

func (self *ESEContext) GetTaggedValueFlags(tagData uint16) uint16 {
	return tagData & (^self.MaskIb())
}
