# go-sml: Idiomatic Go SML Parser

## Overview

A Go library for parsing SML (Smart Message Language) data from German smart meters. Parsing only — no serialization. Designed to feel like a native Go library, not a C port.

**Module:** `github.com/databus23/go-sml`
**Go version:** 1.22+
**External dependencies:** none (stdlib only)

## Background

SML is a binary protocol specified by VDE's FNN for reading smart electricity meters in Germany. Data is TLV-encoded and transported over serial interfaces (typically 9600 baud, 8-N-1). An SML transmission ("file") contains a sequence of messages framed by escape sequences with CRC16 checksums.

Reference implementation: [volkszaehler/libsml](https://github.com/volkszaehler/libsml) (C99, ~27 source files).

## Architecture

**Approach B: Multi-package with internal core.**

The root `sml` package contains all types, parsing logic, and convenience APIs. A `transport` sub-package handles frame detection from raw byte streams (serial ports).

```
sml (root)           transport (sub-package)
┌──────────────┐     ┌──────────────────┐
│ Types        │     │ Frame detection   │
│ TLV decoder  │     │ Escape handling   │
│ Msg parsers  │     │ CRC validation    │
│ Convenience  │     │ io.Reader wrapper │
│ Listen API   │     └──────────────────┘
└──────────────┘           │
       ▲                   │ returns []byte
       │                   │
       └───────────────────┘
```

The transport package returns raw bytes — it does not import the root package. Users who already have framed data (e.g., from `.bin` files) use `sml.Decode()` directly without touching the transport package.

## Core Types

### File & Message

```go
// File is a decoded SML transmission containing one or more messages.
type File struct {
    Messages []*Message
}

// Message is a single SML message within a file.
type Message struct {
    TransactionID []byte
    GroupID       uint8
    AbortOnError  uint8
    Body          MessageBody
    CRC           uint16
}

// MessageBody is implemented by each specific message type.
type MessageBody interface {
    messageTag() uint32
}
```

### Message Body Types

Only response types are parsed (we're reading meters, not sending requests). Request message tags are skipped without error.

```go
type OpenResponse struct {
    Codepage   *string
    ClientID   []byte
    ReqFileID  []byte
    ServerID   []byte
    RefTime    *Time
    SmlVersion *uint8
}

type CloseResponse struct {
    Signature []byte
}

type GetListResponse struct {
    ClientID       []byte
    ServerID       []byte
    ListName       []byte
    ActSensorTime  *Time
    ValList        []ListEntry
    Signature      []byte
    ActGatewayTime *Time
}

type AttentionResponse struct {
    ServerID         []byte
    AttentionNo      []byte
    AttentionMsg     *string
    AttentionDetails []TreeEntry
}

type GetProcParameterResponse struct {
    ServerID      []byte
    TreePath      TreePath   // flat path of OBIS identifiers
    ParameterTree *TreeEntry // recursive parameter tree
}

type GetProfileListResponse struct {
    ServerID        []byte
    ActTime         *Time
    RegPeriod       *uint32
    ParameterTreePath TreePath  // flat path, not a recursive tree
    ValTime         *Time
    Status          *uint64
    ValList         []PeriodEntry
    RawData         []byte
    PeriodSignature []byte
}

type GetProfilePackResponse struct {
    ServerID          []byte
    ActTime           *Time
    RegPeriod         *uint32
    ParameterTreePath TreePath  // flat path, not a recursive tree
    HeaderList        []ProfileObjHeaderEntry
    PeriodList        []ProfileObjPeriodEntry
    RawData           []byte
    PeriodSignature   []byte
}
```

### ListEntry

The primary struct users interact with — represents a single meter reading.

```go
type ListEntry struct {
    ObjName   []byte  // OBIS code (typically 6 bytes)
    Status    *uint64
    ValTime   *Time
    Unit      *uint8  // DLMS unit code
    Scaler    *int8
    Value     Value
    Signature []byte
}
```

### Value

Go interface with concrete named types. Users access via type switch.

```go
type Value interface {
    smlValue()
}

type OctetString []byte
type Bool bool
type Int8 int8
type Int16 int16
type Int32 int32
type Int64 int64
type Uint8 uint8
type Uint16 uint16
type Uint32 uint32
type Uint64 uint64
```

The specific integer size is preserved from the wire encoding (matters for compatibility with the C library's output and for handling meter quirks like the DZG 2-byte negative value case).

### Time

Tagged choice — exactly one variant is represented.

```go
type Time struct {
    Tag   uint8   // 0x01 = SecIndex, 0x02 = Timestamp
    Value uint32
}

const (
    TimeSecIndex  uint8 = 0x01
    TimeTimestamp uint8 = 0x02
)
```

### TreePath

Flat sequence of OBIS identifiers used in proc-parameter and profile requests. Distinct from TreeEntry (recursive).

```go
// TreePath is a flat path of OBIS code segments.
type TreePath [][]byte
```

### TreeEntry

Recursive structure for parameter data.

```go
type TreeEntry struct {
    ParameterName  []byte
    ParameterValue Value  // Note: simplified from C's sml_proc_par_value which can also hold period/tuple/time entries. If test data reveals non-Value variants, this will be revisited.
    Children       []TreeEntry
}
```

### PeriodEntry

Used in GetProfileListResponse.

```go
type PeriodEntry struct {
    ObjName       []byte
    Unit          *uint8
    Scaler        *int8
    Value         Value
    Signature     []byte
}
```

### ProfileObjHeaderEntry and ProfileObjPeriodEntry

Used in GetProfilePackResponse.

```go
type ProfileObjHeaderEntry struct {
    ObjName  []byte
    Unit     *uint8
    Scaler   *int8
}

type ProfileObjPeriodEntry struct {
    ValTime    *Time
    Status     *uint64
    ValueList  []PeriodEntry
}
```

### Design Decisions for Types

- Optional fields use pointers (`*uint8`, `*int8`, `*string`) — nil means absent per SML spec.
- Binary data (`[]byte`) for OBIS codes, server IDs, signatures — not strings.
- Slices instead of linked lists.
- Value as interface + named types instead of tagged union struct — idiomatic Go type switch.
- Parsed values own their backing memory — no shared buffers between the decoder and returned types. Safe to use after the decode call returns.
- `OctetString` (`[]byte`) values are independent copies, not slices of the input buffer.

## TLV Decoder

Internal (unexported) decoder that reads SML's binary type-length-value encoding.

```go
type decoder struct {
    buf []byte
    pos int
    err error
}
```

SML TLV format: each element starts with type-length byte(s). High nibble = type (0x0 octet string, 0x4 boolean, 0x5 signed int, 0x6 unsigned int, 0x7 list), low nibble = length. A "more" bit indicates multi-byte length fields.

**Sticky error pattern:** Once `d.err` is set, all subsequent reads are no-ops. Caller checks `d.err` once after parsing a complete structure. Same pattern as `encoding/binary` and `bufio.Scanner` — avoids `if err != nil` after every TLV read.

**Key methods:**

- `readByte()`, `readTypeLength()` — low-level byte reading
- `isEndOfMessage()`, `isOptionalSkipped()` — marker detection (0x00 and 0x01)
- `readOctetString()`, `readBool()`, `readSigned()`, `readUnsigned()` — type-specific readers
- `readValue()` — dispatches by type tag, returns appropriately-sized Value
- `readListLength()` — reads list header, returns element count
- Optional variants (`readOptionalOctetString()`, etc.) — return pointer/nil

## Message Parsers

Internal functions called by `Decode()`. Each reads fields in sequence using the decoder.

**Top-level flow:**

```go
func Decode(data []byte) (*File, error)
```

Creates a decoder, loops reading messages until end-of-data. Each message:
1. Read transaction ID (octet string)
2. Read group ID (unsigned)
3. Read abort-on-error (unsigned)
4. Read message body — read uint32 tag, dispatch to type-specific parser
5. Read CRC
6. Read end-of-message marker

**Only response types are parsed.** Request tags (0x...00) are skipped. Response tags (0x...01) dispatch to their parser.

**Configurable strictness:**

```go
type DecodeOptions struct {
    Strict bool // default: true — fail on first malformed message
}

func DecodeWithOptions(data []byte, opts DecodeOptions) (*File, error)
```

`Decode()` wraps `DecodeWithOptions` with `Strict: true`.

In non-strict mode, `DecodeWithOptions` returns partial results in `*File` and a combined error via `errors.Join`. Callers can unwrap individual errors with `errors.As`/`errors.Is`.

## Transport Layer

Sub-package `transport` — reads SML frames from an `io.Reader`.

### Framing Protocol

- Start: `1b 1b 1b 1b 01 01 01 01`
- End: `1b 1b 1b 1b 1a XX` (XX = padding byte count), followed by 2-byte CRC
- Escape: `1b 1b 1b 1b` in payload becomes `1b 1b 1b 1b 1b 1b 1b 1b` on wire
- Padding: messages padded to 4-byte alignment

### API

```go
package transport

// Reader reads SML frames from an underlying io.Reader.
type Reader struct { /* unexported fields */ }

func NewReader(r io.Reader) *Reader

// Next returns the next complete SML frame (transport stripped, CRC validated).
// Returns io.EOF when the underlying reader is exhausted.
func (r *Reader) Next() ([]byte, error)
```

**Max frame size:** The reader enforces a default maximum frame size of 64KB to prevent unbounded memory growth on malformed streams. This is configurable but the default should be safe for all known meters (real frames are typically under 1KB).

### CRC Handling

Tries standard CRC16 (DIN 62056-46) first. On failure, tries CRC-16/Kermit (HOLLEY DTZ541 workaround). Auto-detection, no user configuration needed.

### Internal Design

State machine: `scanningForStart` -> `readingPayload` -> `foundEnd` -> `validating`. Simple byte accumulation — SML frames are typically under 1KB.

## Convenience API

Methods on core types for common operations.

```go
// ScaledValue returns the value as float64 with scaler applied (value * 10^scaler).
// Returns ok=false if value is not numeric or scaler is absent.
func (e *ListEntry) ScaledValue() (float64, bool)

// OBISString returns the OBIS code as "A-B:C.D.E*F".
func (e *ListEntry) OBISString() string

// UnitString returns the DLMS unit name (e.g., "Wh", "W", "V").
func (e *ListEntry) UnitString() string

// Readings extracts all ListEntry values from GetListResponse messages.
func (f *File) Readings() []ListEntry
```

## Streaming API

For long-running meter reading sessions, using context for cancellation.

```go
// Handler is called for each decoded SML file. Return non-nil error to stop listening.
type Handler func(*File) error

// MessageHandler is called for each message. Return non-nil error to stop listening.
type MessageHandler func(*Message) error

// Listen reads SML frames from r, decodes them, and calls handler for each file.
// Blocks until r errors, ctx is cancelled, or handler returns a non-nil error.
func Listen(ctx context.Context, r io.Reader, handler Handler) error

// ListenMessages calls handler per-message rather than per-file.
func ListenMessages(ctx context.Context, r io.Reader, handler MessageHandler) error
```

`Listen` composes `transport.NewReader` + `sml.Decode` internally.

**Error resilience:** When a single frame fails CRC or decoding, `Listen` skips it and continues reading the next frame. Only fatal errors (io.Reader failure, context cancellation, handler error) stop the loop. Skipped frame errors are not surfaced — for long-running serial sessions, transient corruption is expected and should not kill the listener.

## DLMS Units

Lookup table mapping unit codes to strings per ISO EN 62056-62. Implemented as a `[256]string` array indexed by unit code for zero-allocation lookup. Covers the standard set (~70 units: Wh, kWh, W, V, A, Hz, var, etc.).

## Meter Quirks

Known meter-specific workarounds:

1. **HOLLEY DTZ541:** Uses CRC-16/Kermit instead of standard CRC16. Handled by auto-detection in transport layer.
2. **DZG DVS-7412:** Wrongly encodes unsigned power values as signed 2-byte integers. The parser preserves the wire encoding faithfully — the Value type will reflect what the meter actually sent (Int16 with a negative value). This matches libsml's behavior for compatibility.

## Package Layout

```
github.com/databus23/go-sml/
├── sml.go              // File, Message, Decode, DecodeWithOptions
├── types.go            // Value, ListEntry, Time, message body types
├── decode.go           // decoder struct, TLV methods
├── messages.go         // readOpenResponse, readGetListResponse, etc.
├── convenience.go      // ScaledValue, OBISString, UnitString, Readings
├── listen.go           // Listen, ListenMessages
├── units.go            // DLMS unit table
├── transport/
│   ├── transport.go    // Reader, NewReader, Next
│   ├── transport_test.go
│   ├── crc.go          // CRC16 DIN 62056-46 + Kermit
│   └── escape.go       // Escape sequence processing
├── testdata/
│   ├── *.bin           // From libsml-testing
│   └── golden/
│       └── *.txt       // Pre-generated sml_server output
├── sml_test.go
├── integration_test.go
├── fuzz_test.go
├── go.mod
```

## Testing Strategy

### Tier 1: Unit Tests

- **TLV decoder:** Hand-crafted byte inputs for each primitive type, multi-byte lengths, optional markers, error cases.
- **Message parsers:** Minimal valid binary representations of each message type.
- **Transport:** Frame detection, escape unescaping, CRC validation (both variants), padding, partial reads, garbage data.
- **Convenience:** ScaledValue with various scaler/value combos, OBISString formatting, UnitString lookup.

### Tier 2: Integration Tests (libsml-testing)

- 38 `.bin` files from [devZer0/libsml-testing](https://github.com/devZer0/libsml-testing) stored in `testdata/`.
- Golden files generated once by running C `sml_server` against each `.bin`, stored in `testdata/golden/`.
- Test: parse each `.bin` with our library, format output to match `sml_server`, compare against golden file.
- Covers 15+ meter models from 8+ manufacturers, including known edge cases (negative values, non-standard CRC, error data).

### Tier 3: Fuzz Testing

- Go native fuzzing (`testing.F`) with `.bin` files as seed corpus.
- Targets: `sml.Decode()`, `transport.Reader`, and the internal TLV decoder — must never panic on any input.

### What We Don't Test

- No mocks. All tests use real byte data.
- No testing of mocked behavior.

## Exported Surface Summary

**Root package `sml`:**
- Types: ~15 (File, Message, MessageBody, Value interface + 10 concrete types, ListEntry, Time, TreeEntry, message body structs, DecodeOptions)
- Functions: Decode, DecodeWithOptions, Listen, ListenMessages
- Methods: ScaledValue, OBISString, UnitString, Readings

**Sub-package `transport`:**
- Types: Reader
- Functions: NewReader
- Methods: Next

Zero external dependencies.
