# go-ese DateTime Decoding — Full Investigation and Fix

## Overview

During forensic analysis of Active Directory Certificate Services (ADCS) CA
databases using Velociraptor's `parse_ese` VQL plugin, all `DateTime` column
fields returned `1899-12-30T00:00:00Z` instead of correct timestamps. This
document captures the full investigation, the failed approaches, and the final
correct fix implemented in `parser/catalog.go`.

---

## Background: ESE DateTime Column Storage

The Extensible Storage Engine (ESE) `DateTime` column type
(`JET_coltypDateTime`, type code `0x08`) can store dates in one of two formats
depending on the application that created the database:

1. **OLE variant double** — a 64-bit IEEE 754 floating-point number
   representing days since December 30, 1899. Used by Windows SRUM
   (`SRUDB.dat`), User Access Logs (`Current.mdb`), and others.

2. **Windows FILETIME** — a 64-bit unsigned integer counting 100-nanosecond
   intervals since January 1, 1601 UTC. Used by ADCS CA databases
   (`CertLog\*.edb`), confirmed by direct binary analysis and by both
   `certutil` and `esedbexport` correctly decoding the same bytes.

The go-ese library previously used the column's `Flags` value from the ESE
catalog to choose the decoder:

- `Flags=1` → Windows FILETIME (`WinFileTime64`)
- `Flags=0` → OLE variant double (`math.Float64frombits` + day arithmetic)

---

## The ADCS Problem

ADCS CA databases store `DateTime` values as Windows FILETIMEs but declare
their columns with `Flags=0` in the catalog. This sent go-ese down the OLE
double path.

A real FILETIME for a date in 2025–2026 is approximately
`134,000,000,000,000,000`. When those 8 bytes are reinterpreted as an IEEE 754
double via `math.Float64frombits()`, the result is a subnormal float on the
order of `10^-299`. Multiplying by `86400` (seconds per day) still yields
essentially zero, so the computed Unix timestamp is always approximately
`-2,209,334,400`, which decodes to `1899-12-30` for every row regardless of
the actual date stored.

This corruption is **irreversible**: every non-zero FILETIME produces the same
near-zero result, so the original value cannot be recovered from go-ese's
output.

### Affected columns in ADCS CA databases

**`Requests` table:** `SubmittedWhen`, `ResolvedWhen`, `RevokedWhen`,
`RevokedEffectiveWhen`

**`Certificates` table:** `NotBefore`, `NotAfter`

### Confirmation by binary analysis

Raw bytes for `SubmittedWhen` of RequestID=1 in `ESSOS-CA.edb`:

```
30 20 6E C1 09 11 DC 01
```

Decoded as a Windows FILETIME: `134000822511870000` → `2025-08-19 13:04:11.187 UTC`

This matches the output of both `certutil` and `esedbexport`. go-ese was
decoding the same bytes as an OLE double, yielding `~6.6e-304` days →
`1899-12-30`.

---

## Investigation — Failed First Approach

The initial fix unconditionally decoded all `DateTime` columns with `Flags=0`
as Windows FILETIMEs, removing the `switch column.Flags` branch entirely. This
was tested and then **reverted** because it broke the SRUM test fixture.

The `SRUDB.dat` test database (`{D10CA2FE-6FCF-4F6D-848E-B2E99266FA86}` table)
has a `TimeStamp` column with identical catalog metadata to the ADCS columns:

- `TypeByte = 0x08` (DateTime)
- `ColumnFlags = 0x00000000` (Flags=0)

But the actual bytes in the SRUDB are genuine OLE variant doubles. For example:

```
2E D8 82 2D A1 B7 E5 40  →  float64 44477.04  →  2021-10-08 00:53:00 UTC
```

The catalog metadata is **identical** for both databases. The encoding is
determined by the application that wrote the data, not by any flag or marker
in the ESE schema. A blanket unconditional fix would correct ADCS but break
SRUM, UAL, and any other database that uses genuine OLE doubles.

---

## The Correct Fix — Self-Discriminating Value Inspection

Although the catalog metadata cannot distinguish the two encodings, the raw
8-byte values themselves occupy completely different regions of the IEEE 754
float64 number space:

| Encoding | Example bytes | Interpreted as float64 | Value range |
|---|---|---|---|
| OLE double (SRUDB, 2021-10-08) | `2E D8 82 2D A1 B7 E5 40` | `44477.04` | `[2.0, ~73050]` — normal float, high byte `0x40` |
| FILETIME (ADCS, 2025-08-19) | `30 20 6E C1 09 11 DC 01` | `~1.05e-299` | `[0, ~10^-295]` — subnormal float, high byte `0x01` |
| Zero / unset | `00 00 00 00 00 00 00 00` | `0.0` | `0` |

**Key insight:**

- Any valid OLE double for a date between 1900 and 2100 produces a normal
  IEEE 754 float in the range `[2.0, ~73050]`. These integers have their
  significant bits in the exponent and high mantissa — the high byte is
  always `0x40`.

- Any valid FILETIME for a date between 1601 and 9999, when its bytes are
  misread as a float64, produces a subnormal number on the order of
  `10^-299`. FILETIME integers (~`1.3e17`) have their significant bits in
  positions that map entirely to the mantissa of a subnormal double — the
  high byte is always `0x01` or lower.

- The gap between `~10^-295` (maximum FILETIME-as-float) and `2.0` (minimum
  valid OLE double) spans many hundreds of orders of magnitude. There is no
  overlap and no ambiguity.

**Threshold:** `if float64(raw_bytes) > 1.0` → OLE double. Otherwise →
Windows FILETIME.

### Implementation

In the `Flags=0` branch of the `DateTime` decoder (both fixed-column and
tagged-column sections of `tagToRecord` in `parser/catalog.go`):

```go
case 0:
    value_int := ParseUint64(reader, offset)
    days_since_1900 := math.Float64frombits(value_int)

    if days_since_1900 > 1.0 {
        // Genuine OLE variant double encoding.
        result.Set(column.Name,
            time.Unix(int64(days_since_1900*24*60*60)+
                -2208988800-2*24*60*60, 0).UTC())
    } else {
        // Windows FILETIME encoding (e.g. ADCS CA databases).
        result.Set(column.Name, WinFileTime64(reader, offset))
    }
```

The same logic applies in the tagged-column section using
`WinFileTime64Bin(bytes)` for the FILETIME path.

---

## Validation

### ADCS CA database (`ESSOS-CA.edb`)

| RequestID | Column | Raw bytes | Previous result | Fixed result |
|---|---|---|---|---|
| 1 | `SubmittedWhen` | `30 20 6E C1 09 11 DC 01` | `1899-12-30T00:00:00Z` | `2025-08-19T13:04:11Z` |
| 2 | `SubmittedWhen` | `20 DC 98 CC 09 11 DC 01` | `1899-12-30T00:00:00Z` | `2025-08-19T13:04:29Z` |
| 3 | `SubmittedWhen` | `20 3B 91 BD 9B D1 DC 01` | `1899-12-30T00:00:00Z` | `2026-04-21T14:32:54Z` |

### SRUM database (`SRUDB.dat`) — must remain correct

| Column | Raw bytes | float64 interpretation | Result |
|---|---|---|---|
| `TimeStamp` | `2E D8 82 2D A1 B7 E5 40` | `44477.04` (> 1.0 → OLE double) | `2021-10-08T00:53:00Z` ✓ |

The go-ese test suite passes without modification to any golden fixture files.

---

## Files Changed

| File | Change |
|---|---|
| `parser/catalog.go` | Added `if days_since_1900 > 1.0` branch inside the `Flags=0` DateTime decoder in both the fixed-column and tagged-column sections |
| `Changes.md` | This document |

No other files were modified. The `"time"` and `"math"` imports are both
retained as they are still used after the fix.

---

## Additional Context: The VQL Workaround

Before this library fix was developed, a workaround was implemented at the
Velociraptor VQL artifact level (`Custom.Windows.ADCS.CertificateAuthority.EDB`).
That workaround copies the EDB to a temp file, patches the `DateTime` column
type bytes from `0x08` to `0x0F` (LongLong) in the catalog, then runs
`parse_ese` on the patched copy. This forces go-ese to return the raw `uint64`
FILETIME values which are decoded with `timestamp(winfiletime=...)` in VQL.

With this library fix in place that workaround is no longer necessary. The
VQL artifact can read `DateTime` columns directly without any catalog patching.
