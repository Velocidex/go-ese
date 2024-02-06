# ESE Format observations

Previously this library was based on libesedb which was an excellent
effort to reverse engineer the file format. Since then, Microsoft has
published the source code for the ESE engine. This clarifies a lot of
questions about the file format.

These are described and implemented here
https://github.com/microsoft/Extensible-Storage-Engine/blob/main/dev/ese/src/ese/lv.cxx

TODO: This library currently uses variable names based on libesedb but
since we now have the original Microsoft source code it makes sense to
change the names to make them the same as the original source code.

## Small pages

From Extensible-Storage-Engine/dev/ese/src/inc/daedef.hxx#L5631 If
page size is less than 8kb then it is considered a small page. There
are a number of changes throughout the format for small page vs large
page support.

### B trees

The pages are organized into B trees so we can walk them.

Example Long Value page from `ntds.dit`. In this case the page size is 0x2000

```
00FD0000   AC CD 8E BF 3E 02 C1 FD  26 3E 01 00 00 00 00 00  ....>...&>......
00FD0010   00 00 00 00 E8 07 00 00  E4 00 00 00 5B 02 00 00  ............[...
00FD0020   5D 1D 08 00 82 A8 00 00  00 00 00 01 00 00 00 00  ]...............
00FD0030   04 00 00 00 01 00 00 00  74 0F 00 00 08 00 00 00  ........t.......
```

The page starts with a PageHeader struct. This has two versions:

https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/inc/cpage.hxx#L885

PGHDR used for pages less than 8kb (small pages) has a size of 40 bytes
PGHDR2 has a size of 80 bytes for larger pages.

The size of the header depends on the pagesize on which version of header is used.
https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/inc/cpage.hxx#L987


// Todo - these fields should be renamed to correspond with the MS
// source code.

```
Page header struct PageHeader @ 0xfd0000:
  LastModified: {
  struct DBTime @ 0xfd0008:
    Hours: 0x3e26
    Min: 0x1
    Sec: 0x0
  }
  PreviousPageNumber: 0x0
  NextPageNumber: 0x7e8
  FatherPage: 0xe4
  AvailableDataSize: 0x25b
  AvailableDataOffset: 0x1d5d
  AvailablePageTag: 0x8
  Flags: 43138 (Leaf,Long)
  EndOffset: 0xfd0028   = 0xfd0000 + 40
```

Immediately following the PageHeader we have the external_value:
  00 00 00 01 00 00 00 00

This is marked by the first tag.

### Tags

The data is stored in the page in a list of values. The index to these
values is stored in a list of Tags at the end of the page.

```
00FD1FD0   00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  ................
00FD1FE0   0D 00 50 9D E1 06 6F 96  0D 00 62 96 C9 06 99 8F  ..P...o...b.....
00FD1FF0   0D 00 8C 8F 78 0F 14 80  0C 00 08 80 08 00 00 00  ....x...........
```

The first tag is always reserved for the "external header". In this
case it is pointing at a value of length 8 bytes immediately after the
page header (Tag offset 0)

The next tag is decoded as:
```
ID 2023 Tag struct Tag @ 0xfd1ff8:
  _ValueSize: 0xc
  _ValueOffset: 0x8008
   ValueOffsetInPage: 0xfd0030
 PageID 2023 Flags 4
Header struct LongValueHeader @ 0xfd0030:
  Flags: Unknown (4)
  Key: ^A^@^@^@
```

To calculate the offset within the page (ValueOffsetInPage) we mask
_ValueOffset with 0x7FFF for large pages and 0x1fff for small
pages. This is the offset **after** the end of the header - so in the
above the value's offset is:

```
0x8008 & 0x1FFF = 8
8 + 40 + 0xfd0000 = 0xfd0030
```

See the following calculations:

https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/ese/cpage.cxx#L1200
https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/inc/cpage.hxx#L840
(In the above code Ib - index of buffer, Cb means count buffer)

The value is stored within the page. This contains the Key and data.
```
00FD0030   04 00 00 00 01 00 00 00  74 0F 00 00 08 00 00 00  ........t.......
00FD0040   01 00 14 8C 54 0F 00 00  64 0F 00 00 14 00 00 00  ....T...d.......
00FD0050   4C 01 00 00 04 00 38 01  07 00 00 00 07 42 38 00  L.....8......B8.
```

### Data Records

The data fields are stored in a LINE (or node) within the page using
Tags as above. But the data stores a series of REC structs. These are
defined in rec.hxx#284

The REC starts with a header then a sequence of fixed length fields
(for columns with a fixed size like ints etc), then variable length
fields (like Text), followed by tagged fields.

RECHDR:
   BYTE fidFixedLastInRec
   BYTE fidVarLastInRec
   USHORT ibEndOfFixedData   // offset relative to start of record

Here we see it being modified fldmod.cxx#895

### Long Value Key

A key has a prefix and a suffix. It is defined in
https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/inc/daedef.hxx#L1248

The key is reversed on LE platforms and is stored in Big Endian on
disk.

There are two types of keys. The key can be 32 bit (8 byte) or 64 bit
(12 byte).

### Line

The ESE code calls the Value within the page a `Line`.

A Line is a Tagged Value in the page.

struct LINE {
       pv -> pointer to the Value data
       cb -> size of the value data
       fFlags -> tag flags
}

// https://github.com/microsoft/Extensible-Storage-Engine/blob/933dc839b5a97b9a5b3e04824bdd456daf75a57d/dev/ese/src/inc/node.hxx#L226
tag flags can be fNDCompressed = 0x04;

To get a line from the page call cpage.GetPtr()

Extracting the key from the line: NDILineToKeydataflags

The Key has a prefix and suffix part. If the tag fFlags is compressed
(bit 16 is on) then prefix count is the first 4 bytes and it refers to
the external header for the actual data.

Otherwise, prefix is not used and suffix count is the first 4 bytes
and suffix buffer is the next 4 bytes.

Examples:

1. Key is not compressed 0400000001000000740f0000
   Key Prefix : 000001  (Length 0400)
   Key Suffix : "" (Length 0000)
   Total Key: 00 00 00 00 00 00 00 01
   Data: 000000740f0000

2. Key is compressed 080000000100148c540f0000 (external header is 00 00 00 01 00 00 00)
   Key Prefix: Length 8 - value is in the external header 00000001000000
   Key Suffix: Length (0000 ) - next 4 bytes (so no local header).
   Total key: 00 00 00 01 00 00 00 00
   Data: 0100148c540f0000

3. Key is compressed 0300050002000000000100148ca006
   Key Prefix: Length 3 Value in external header (00 00 00)
   Key Suffix: Length 5. Value is 02 00 00 00 00
   Total Key: 00 00 00 02 00 00 00 00
   Data: 0100148ca006
