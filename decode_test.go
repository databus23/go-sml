package sml

import (
	"bytes"
	"io"
	"testing"
)

func TestDecoderReadByte(t *testing.T) {
	t.Run("reads single byte and advances position", func(t *testing.T) {
		d := newDecoder([]byte{0xAB, 0xCD})
		b := d.readByte()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if b != 0xAB {
			t.Fatalf("got 0x%02X, want 0xAB", b)
		}
		if d.pos != 1 {
			t.Fatalf("pos = %d, want 1", d.pos)
		}

		b = d.readByte()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if b != 0xCD {
			t.Fatalf("got 0x%02X, want 0xCD", b)
		}
		if d.pos != 2 {
			t.Fatalf("pos = %d, want 2", d.pos)
		}
	})

	t.Run("error on EOF", func(t *testing.T) {
		d := newDecoder([]byte{0x42})
		_ = d.readByte()
		b := d.readByte()
		if d.err != io.ErrUnexpectedEOF {
			t.Fatalf("err = %v, want io.ErrUnexpectedEOF", d.err)
		}
		if b != 0 {
			t.Fatalf("got 0x%02X, want 0x00 on error", b)
		}
	})

	t.Run("error on empty buffer", func(t *testing.T) {
		d := newDecoder([]byte{})
		b := d.readByte()
		if d.err != io.ErrUnexpectedEOF {
			t.Fatalf("err = %v, want io.ErrUnexpectedEOF", d.err)
		}
		if b != 0 {
			t.Fatalf("got 0x%02X, want 0x00 on error", b)
		}
	})
}

func TestDecoderReadTypeLength(t *testing.T) {
	t.Run("single-byte TL unsigned int length 5", func(t *testing.T) {
		// 0x65 = 0b0_110_0101 → more=0, type=0x60 (unsigned), length nibble=5
		// Non-list: data_len = 5 - 1 (one TL byte) = 4
		d := newDecoder([]byte{0x65, 0x00, 0x00, 0x00, 0x00})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x60 {
			t.Fatalf("type = 0x%02X, want 0x60", typ)
		}
		if length != 4 {
			t.Fatalf("length = %d, want 4", length)
		}
		if d.pos != 1 {
			t.Fatalf("pos = %d, want 1 (only TL byte consumed)", d.pos)
		}
	})

	t.Run("single-byte TL octet string length 3", func(t *testing.T) {
		// 0x03 = 0b0_000_0011 → more=0, type=0x00 (octet string), length nibble=3
		// Non-list: data_len = 3 - 1 = 2
		d := newDecoder([]byte{0x03, 0xAA, 0xBB})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x00 {
			t.Fatalf("type = 0x%02X, want 0x00", typ)
		}
		if length != 2 {
			t.Fatalf("length = %d, want 2", length)
		}
	})

	t.Run("single-byte TL boolean", func(t *testing.T) {
		// 0x42 = 0b0_100_0010 → more=0, type=0x40 (boolean), length nibble=2
		// Non-list: data_len = 2 - 1 = 1
		d := newDecoder([]byte{0x42, 0x01})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x40 {
			t.Fatalf("type = 0x%02X, want 0x40", typ)
		}
		if length != 1 {
			t.Fatalf("length = %d, want 1", length)
		}
	})

	t.Run("single-byte TL signed int", func(t *testing.T) {
		// 0x53 = 0b0_101_0011 → more=0, type=0x50 (signed int), length nibble=3
		// Non-list: data_len = 3 - 1 = 2
		d := newDecoder([]byte{0x53, 0xFF, 0xFE})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x50 {
			t.Fatalf("type = 0x%02X, want 0x50", typ)
		}
		if length != 2 {
			t.Fatalf("length = %d, want 2", length)
		}
	})

	t.Run("single-byte TL list with 3 elements", func(t *testing.T) {
		// 0x73 = 0b0_111_0011 → more=0, type=0x70 (list), length nibble=3
		// List: length IS element count = 3
		d := newDecoder([]byte{0x73})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x70 {
			t.Fatalf("type = 0x%02X, want 0x70", typ)
		}
		if length != 3 {
			t.Fatalf("length = %d, want 3", length)
		}
	})

	t.Run("multi-byte TL with more bit", func(t *testing.T) {
		// First byte: 0x83 = 0b1_000_0011 → more=1, type=0x00 (octet string), length nibble=3
		// Second byte: 0x12 = 0b0_001_0010 → more=0, low nibble=2
		// Combined length: (3 << 4) | 2 = 50
		// Non-list: data_len = 50 - 2 (two TL bytes) = 48
		d := newDecoder(append([]byte{0x83, 0x12}, make([]byte, 48)...))
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x00 {
			t.Fatalf("type = 0x%02X, want 0x00", typ)
		}
		if length != 48 {
			t.Fatalf("length = %d, want 48", length)
		}
		if d.pos != 2 {
			t.Fatalf("pos = %d, want 2 (two TL bytes consumed)", d.pos)
		}
	})

	t.Run("multi-byte TL list with more bit", func(t *testing.T) {
		// First byte: 0xF1 = 0b1_111_0001 → more=1, type=0x70 (list), length nibble=1
		// Second byte: 0x02 = 0b0_000_0010 → more=0, low nibble=2
		// Combined length: (1 << 4) | 2 = 18
		// List: length IS element count = 18
		d := newDecoder([]byte{0xF1, 0x02})
		typ, length := d.readTypeLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if typ != 0x70 {
			t.Fatalf("type = 0x%02X, want 0x70", typ)
		}
		if length != 18 {
			t.Fatalf("length = %d, want 18", length)
		}
	})

	t.Run("error on EOF during multi-byte TL", func(t *testing.T) {
		// First byte has more bit set but no second byte
		d := newDecoder([]byte{0x83})
		_, _ = d.readTypeLength()
		if d.err != io.ErrUnexpectedEOF {
			t.Fatalf("err = %v, want io.ErrUnexpectedEOF", d.err)
		}
	})
}

func TestDecoderIsEndOfMessage(t *testing.T) {
	t.Run("returns true for 0x00 without consuming", func(t *testing.T) {
		d := newDecoder([]byte{0x00, 0xFF})
		if !d.isEndOfMessage() {
			t.Fatal("expected true for 0x00")
		}
		if d.pos != 0 {
			t.Fatalf("pos = %d, want 0 (should not consume)", d.pos)
		}
	})

	t.Run("returns false for non-zero byte", func(t *testing.T) {
		d := newDecoder([]byte{0x65, 0x00})
		if d.isEndOfMessage() {
			t.Fatal("expected false for 0x65")
		}
		if d.pos != 0 {
			t.Fatalf("pos = %d, want 0 (should not consume)", d.pos)
		}
	})

	t.Run("returns false on empty buffer", func(t *testing.T) {
		d := newDecoder([]byte{})
		if d.isEndOfMessage() {
			t.Fatal("expected false on empty buffer")
		}
	})

	t.Run("returns false when error already set", func(t *testing.T) {
		d := newDecoder([]byte{0x00})
		d.err = io.ErrUnexpectedEOF
		if d.isEndOfMessage() {
			t.Fatal("expected false when error is set")
		}
	})
}

func TestDecoderIsOptionalSkipped(t *testing.T) {
	t.Run("returns true for 0x01 and consumes it", func(t *testing.T) {
		d := newDecoder([]byte{0x01, 0xFF})
		if !d.isOptionalSkipped() {
			t.Fatal("expected true for 0x01")
		}
		if d.pos != 1 {
			t.Fatalf("pos = %d, want 1 (should consume the byte)", d.pos)
		}
	})

	t.Run("returns false for non-0x01 byte without consuming", func(t *testing.T) {
		d := newDecoder([]byte{0x65, 0x00})
		if d.isOptionalSkipped() {
			t.Fatal("expected false for 0x65")
		}
		if d.pos != 0 {
			t.Fatalf("pos = %d, want 0 (should not consume)", d.pos)
		}
	})

	t.Run("returns false on empty buffer", func(t *testing.T) {
		d := newDecoder([]byte{})
		if d.isOptionalSkipped() {
			t.Fatal("expected false on empty buffer")
		}
	})

	t.Run("returns false when error already set", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		d.err = io.ErrUnexpectedEOF
		if d.isOptionalSkipped() {
			t.Fatal("expected false when error is set")
		}
	})
}

func TestDecoderStickyError(t *testing.T) {
	t.Run("readByte is no-op after error", func(t *testing.T) {
		d := newDecoder([]byte{})
		_ = d.readByte() // triggers EOF
		origErr := d.err

		b := d.readByte()
		if b != 0 {
			t.Fatalf("got 0x%02X, want 0x00 on sticky error", b)
		}
		if d.err != origErr {
			t.Fatalf("error changed from %v to %v", origErr, d.err)
		}
	})

	t.Run("readTypeLength is no-op after error", func(t *testing.T) {
		d := newDecoder([]byte{})
		_ = d.readByte() // triggers EOF

		typ, length := d.readTypeLength()
		if typ != 0 || length != 0 {
			t.Fatalf("got type=0x%02X length=%d, want 0x00/0 on sticky error", typ, length)
		}
	})

	t.Run("error does not change after being set", func(t *testing.T) {
		d := newDecoder([]byte{0x42})
		_ = d.readByte()
		_ = d.readByte() // triggers EOF

		firstErr := d.err
		_ = d.readByte() // should be no-op
		_ = d.readByte() // should be no-op

		if d.err != firstErr {
			t.Fatalf("sticky error changed: got %v, want %v", d.err, firstErr)
		}
	})
}

func TestDecoderReadOctetString(t *testing.T) {
	t.Run("reads 4 data bytes", func(t *testing.T) {
		// 0x05 = type 0x00 (octet string), total length 5, data = 5-1 = 4 bytes
		input := []byte{0x05, 0xDE, 0xAD, 0xBE, 0xEF}
		d := newDecoder(input)
		got := d.readOctetString()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		if !bytes.Equal(got, want) {
			t.Fatalf("got %X, want %X", got, want)
		}
	})

	t.Run("returns a copy not a slice of the input buffer", func(t *testing.T) {
		input := []byte{0x05, 0xDE, 0xAD, 0xBE, 0xEF}
		d := newDecoder(input)
		got := d.readOctetString()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		// Mutate the returned slice
		got[0] = 0xFF
		// Original buffer must be unchanged
		if input[1] != 0xDE {
			t.Fatal("readOctetString returned a slice of the input buffer, not a copy")
		}
	})

	t.Run("reads empty octet string", func(t *testing.T) {
		// 0x01 total length = 1, data = 0 bytes
		d := newDecoder([]byte{0x01})
		got := d.readOctetString()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if len(got) != 0 {
			t.Fatalf("got len %d, want 0", len(got))
		}
	})

	t.Run("error on wrong type", func(t *testing.T) {
		// 0x62 = unsigned type, not octet string
		d := newDecoder([]byte{0x62, 0x42})
		_ = d.readOctetString()
		if d.err == nil {
			t.Fatal("expected error for wrong type, got nil")
		}
	})
}

func TestDecoderReadBool(t *testing.T) {
	t.Run("reads true", func(t *testing.T) {
		// 0x42 = type 0x40 (boolean), total length 2, data = 1 byte
		d := newDecoder([]byte{0x42, 0x01})
		got := d.readBool()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != true {
			t.Fatalf("got %v, want true", got)
		}
	})

	t.Run("reads false", func(t *testing.T) {
		d := newDecoder([]byte{0x42, 0x00})
		got := d.readBool()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != false {
			t.Fatalf("got %v, want false", got)
		}
	})

	t.Run("nonzero is true", func(t *testing.T) {
		d := newDecoder([]byte{0x42, 0xFF})
		got := d.readBool()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != true {
			t.Fatalf("got %v, want true for 0xFF", got)
		}
	})

	t.Run("error on wrong type", func(t *testing.T) {
		// 0x62 = unsigned, not boolean
		d := newDecoder([]byte{0x62, 0x01})
		_ = d.readBool()
		if d.err == nil {
			t.Fatal("expected error for wrong type, got nil")
		}
	})
}

func TestDecoderReadUnsigned(t *testing.T) {
	t.Run("u8", func(t *testing.T) {
		// 0x62 = type 0x60, total length 2, data = 1 byte
		d := newDecoder([]byte{0x62, 0x2A})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint8(0x2A) {
			t.Fatalf("got %v (%T), want Uint8(0x2A)", got, got)
		}
	})

	t.Run("u16", func(t *testing.T) {
		// 0x63 = type 0x60, total length 3, data = 2 bytes
		d := newDecoder([]byte{0x63, 0x01, 0x00})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint16(256) {
			t.Fatalf("got %v (%T), want Uint16(256)", got, got)
		}
	})

	t.Run("u32", func(t *testing.T) {
		// 0x65 = type 0x60, total length 5, data = 4 bytes
		d := newDecoder([]byte{0x65, 0x00, 0x01, 0x00, 0x00})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint32(65536) {
			t.Fatalf("got %v (%T), want Uint32(65536)", got, got)
		}
	})

	t.Run("u64", func(t *testing.T) {
		// 0x69 = type 0x60, total length 9, data = 8 bytes
		d := newDecoder([]byte{0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint64(0x100000000) {
			t.Fatalf("got %v (%T), want Uint64(0x100000000)", got, got)
		}
	})

	t.Run("3 byte unsigned rounds up to u32", func(t *testing.T) {
		// 0x64 = type 0x60, total length 4, data = 3 bytes
		// 0x01 0x00 0x00 = 65536 as u32
		d := newDecoder([]byte{0x64, 0x01, 0x00, 0x00})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint32(65536) {
			t.Fatalf("got %v (%T), want Uint32(65536)", got, got)
		}
	})

	t.Run("5 byte unsigned rounds up to u64", func(t *testing.T) {
		// 0x66 = type 0x60, total length 6, data = 5 bytes
		d := newDecoder([]byte{0x66, 0x01, 0x00, 0x00, 0x00, 0x00})
		got := d.readUnsigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint64(0x100000000) {
			t.Fatalf("got %v (%T), want Uint64(0x100000000)", got, got)
		}
	})

	t.Run("error on wrong type", func(t *testing.T) {
		// 0x52 = signed int, not unsigned
		d := newDecoder([]byte{0x52, 0x01})
		_ = d.readUnsigned()
		if d.err == nil {
			t.Fatal("expected error for wrong type, got nil")
		}
	})
}

func TestDecoderReadSigned(t *testing.T) {
	t.Run("i8 positive", func(t *testing.T) {
		// 0x52 = type 0x50, total length 2, data = 1 byte
		d := newDecoder([]byte{0x52, 0x2A})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int8(42) {
			t.Fatalf("got %v (%T), want Int8(42)", got, got)
		}
	})

	t.Run("i8 negative", func(t *testing.T) {
		// 0xFF as signed i8 = -1
		d := newDecoder([]byte{0x52, 0xFF})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int8(-1) {
			t.Fatalf("got %v (%T), want Int8(-1)", got, got)
		}
	})

	t.Run("i16", func(t *testing.T) {
		// 0x53 = type 0x50, total length 3, data = 2 bytes
		// 0xFF 0xFE = -2 as int16
		d := newDecoder([]byte{0x53, 0xFF, 0xFE})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int16(-2) {
			t.Fatalf("got %v (%T), want Int16(-2)", got, got)
		}
	})

	t.Run("i32", func(t *testing.T) {
		// 0x55 = type 0x50, total length 5, data = 4 bytes
		d := newDecoder([]byte{0x55, 0xFF, 0xFF, 0xFF, 0xFE})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int32(-2) {
			t.Fatalf("got %v (%T), want Int32(-2)", got, got)
		}
	})

	t.Run("i64", func(t *testing.T) {
		// 0x59 = type 0x50, total length 9, data = 8 bytes
		d := newDecoder([]byte{0x59, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int64(-2) {
			t.Fatalf("got %v (%T), want Int64(-2)", got, got)
		}
	})

	t.Run("3 byte signed negative rounds up to i32", func(t *testing.T) {
		// 0x54 = type 0x50, total length 4, data = 3 bytes
		// 0xFF 0xFF 0xFE: high bit set, so sign-extend with 0xFF → 0xFFFF FFFE = -2
		d := newDecoder([]byte{0x54, 0xFF, 0xFF, 0xFE})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int32(-2) {
			t.Fatalf("got %v (%T), want Int32(-2)", got, got)
		}
	})

	t.Run("3 byte signed positive rounds up to i32", func(t *testing.T) {
		// 0x54 = type 0x50, total length 4, data = 3 bytes
		// 0x01 0x00 0x00: high bit not set, pad with 0x00 → 0x0001 0000 = 65536
		d := newDecoder([]byte{0x54, 0x01, 0x00, 0x00})
		got := d.readSigned()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int32(65536) {
			t.Fatalf("got %v (%T), want Int32(65536)", got, got)
		}
	})

	t.Run("error on wrong type", func(t *testing.T) {
		// 0x62 = unsigned, not signed
		d := newDecoder([]byte{0x62, 0x01})
		_ = d.readSigned()
		if d.err == nil {
			t.Fatal("expected error for wrong type, got nil")
		}
	})
}

func TestDecoderReadValue(t *testing.T) {
	t.Run("dispatches octet string", func(t *testing.T) {
		d := newDecoder([]byte{0x05, 0xDE, 0xAD, 0xBE, 0xEF})
		got := d.readValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		os, ok := got.(OctetString)
		if !ok {
			t.Fatalf("got %T, want OctetString", got)
		}
		if !bytes.Equal(os, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
			t.Fatalf("got %X, want DEADBEEF", []byte(os))
		}
	})

	t.Run("dispatches boolean", func(t *testing.T) {
		d := newDecoder([]byte{0x42, 0x01})
		got := d.readValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		b, ok := got.(Bool)
		if !ok {
			t.Fatalf("got %T, want Bool", got)
		}
		if b != Bool(true) {
			t.Fatalf("got %v, want true", b)
		}
	})

	t.Run("dispatches unsigned u16", func(t *testing.T) {
		d := newDecoder([]byte{0x63, 0x01, 0x00})
		got := d.readValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint16(256) {
			t.Fatalf("got %v (%T), want Uint16(256)", got, got)
		}
	})

	t.Run("dispatches signed i8", func(t *testing.T) {
		d := newDecoder([]byte{0x52, 0xFF})
		got := d.readValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Int8(-1) {
			t.Fatalf("got %v (%T), want Int8(-1)", got, got)
		}
	})

	t.Run("error on list type", func(t *testing.T) {
		d := newDecoder([]byte{0x72})
		_ = d.readValue()
		if d.err == nil {
			t.Fatal("expected error for list type in readValue, got nil")
		}
	})
}

func TestDecoderReadListLength(t *testing.T) {
	t.Run("list of 2 elements", func(t *testing.T) {
		// 0x72 = type 0x70 (list), element count = 2
		d := newDecoder([]byte{0x72})
		got := d.readListLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != 2 {
			t.Fatalf("got %d, want 2", got)
		}
	})

	t.Run("list of 7 elements", func(t *testing.T) {
		// 0x77 = type 0x70 (list), element count = 7
		d := newDecoder([]byte{0x77})
		got := d.readListLength()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != 7 {
			t.Fatalf("got %d, want 7", got)
		}
	})

	t.Run("error on wrong type", func(t *testing.T) {
		// 0x62 = unsigned, not list
		d := newDecoder([]byte{0x62, 0x01})
		_ = d.readListLength()
		if d.err == nil {
			t.Fatal("expected error for non-list type, got nil")
		}
	})
}

func TestDecoderReadOptionalOctetString(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		// 0x01 = optional absent
		d := newDecoder([]byte{0x01})
		got := d.readOptionalOctetString()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", got)
		}
	})

	t.Run("present returns data", func(t *testing.T) {
		d := newDecoder([]byte{0x03, 0xAA, 0xBB})
		got := d.readOptionalOctetString()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		want := []byte{0xAA, 0xBB}
		if !bytes.Equal(got, want) {
			t.Fatalf("got %X, want %X", got, want)
		}
	})
}

func TestDecoderReadOptionalValue(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", got)
		}
	})

	t.Run("present returns value", func(t *testing.T) {
		d := newDecoder([]byte{0x62, 0x2A})
		got := d.readOptionalValue()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != Uint8(0x2A) {
			t.Fatalf("got %v (%T), want Uint8(0x2A)", got, got)
		}
	})
}

func TestDecoderReadTime(t *testing.T) {
	t.Run("2-element list with tag and value", func(t *testing.T) {
		// 0x72 = list of 2 elements
		// 0x62 0x01 = u8(1) = TimeSecIndex
		// 0x65 0x00 0x00 0x30 0x39 = u32(12345)
		input := []byte{0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39}
		d := newDecoder(input)
		got := d.readTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got.Tag != TimeSecIndex {
			t.Fatalf("Tag = %d, want %d (TimeSecIndex)", got.Tag, TimeSecIndex)
		}
		if got.Value != 12345 {
			t.Fatalf("Value = %d, want 12345", got.Value)
		}
	})

	t.Run("timestamp tag", func(t *testing.T) {
		// 0x72 = list of 2
		// 0x62 0x02 = u8(2) = TimeTimestamp
		// 0x65 0x00 0x01 0x00 0x00 = u32(65536)
		input := []byte{0x72, 0x62, 0x02, 0x65, 0x00, 0x01, 0x00, 0x00}
		d := newDecoder(input)
		got := d.readTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got.Tag != TimeTimestamp {
			t.Fatalf("Tag = %d, want %d (TimeTimestamp)", got.Tag, TimeTimestamp)
		}
		if got.Value != 65536 {
			t.Fatalf("Value = %d, want 65536", got.Value)
		}
	})
}

func TestDecoderReadTimeBareUnsigned(t *testing.T) {
	t.Run("Holley workaround bare u32", func(t *testing.T) {
		// A bare unsigned integer (no list wrapper). Peek byte 0x65 has type 0x60,
		// not 0x70, so the Holley workaround treats it as SecIndex.
		// 0x65 0x00 0x00 0x30 0x39 = u32(12345)
		input := []byte{0x65, 0x00, 0x00, 0x30, 0x39}
		d := newDecoder(input)
		got := d.readTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got.Tag != TimeSecIndex {
			t.Fatalf("Tag = %d, want %d (TimeSecIndex)", got.Tag, TimeSecIndex)
		}
		if got.Value != 12345 {
			t.Fatalf("Value = %d, want 12345", got.Value)
		}
	})

	t.Run("Holley workaround bare u16", func(t *testing.T) {
		// 0x63 0x00 0x64 = u16(100)
		input := []byte{0x63, 0x00, 0x64}
		d := newDecoder(input)
		got := d.readTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got.Tag != TimeSecIndex {
			t.Fatalf("Tag = %d, want %d (TimeSecIndex)", got.Tag, TimeSecIndex)
		}
		if got.Value != 100 {
			t.Fatalf("Value = %d, want 100", got.Value)
		}
	})
}

func TestDecoderReadTreePath(t *testing.T) {
	t.Run("list of 3 octet strings", func(t *testing.T) {
		// 0x73 = list of 3
		// 0x03 0x01 0x02 = octet string [0x01, 0x02]
		// 0x03 0x03 0x04 = octet string [0x03, 0x04]
		// 0x03 0x05 0x06 = octet string [0x05, 0x06]
		input := []byte{
			0x73,
			0x03, 0x01, 0x02,
			0x03, 0x03, 0x04,
			0x03, 0x05, 0x06,
		}
		d := newDecoder(input)
		got := d.readTreePath()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		want := TreePath{
			{0x01, 0x02},
			{0x03, 0x04},
			{0x05, 0x06},
		}
		for i := range want {
			if !bytes.Equal(got[i], want[i]) {
				t.Fatalf("got[%d] = %X, want %X", i, got[i], want[i])
			}
		}
	})

	t.Run("empty list", func(t *testing.T) {
		// 0x70 = list of 0 elements (but that's unusual; let's test the boundary)
		// Actually 0x70 encodes type=0x70, length=0. This is valid.
		d := newDecoder([]byte{0x70})
		got := d.readTreePath()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})
}

func TestDecoderReadTreeEntry(t *testing.T) {
	t.Run("leaf entry with no children", func(t *testing.T) {
		// 0x73 = list of 3
		// parameter_name: 0x03 0xAA 0xBB (octet string [0xAA, 0xBB])
		// parameter_value: 0x62 0x2A (u8(42))
		// child_list: 0x01 (optional absent)
		input := []byte{
			0x73,
			0x03, 0xAA, 0xBB,
			0x62, 0x2A,
			0x01,
		}
		d := newDecoder(input)
		got := d.readTreeEntry()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if !bytes.Equal(got.ParameterName, []byte{0xAA, 0xBB}) {
			t.Fatalf("ParameterName = %X, want AABB", got.ParameterName)
		}
		if got.ParameterValue != Uint8(42) {
			t.Fatalf("ParameterValue = %v (%T), want Uint8(42)", got.ParameterValue, got.ParameterValue)
		}
		if got.Children != nil {
			t.Fatalf("Children = %v, want nil", got.Children)
		}
	})

	t.Run("entry with children", func(t *testing.T) {
		// Parent: list of 3
		//   parameter_name: 0x03 0x01 0x02
		//   parameter_value: 0x01 (optional absent)
		//   child_list: list of 1 child
		//     child: list of 3
		//       parameter_name: 0x03 0x03 0x04
		//       parameter_value: 0x62 0xFF
		//       child_list: 0x01 (optional absent)
		input := []byte{
			0x73,             // parent list of 3
			0x03, 0x01, 0x02, // parameter_name
			0x01,             // parameter_value (absent)
			0x71,             // child_list: list of 1
			0x73,             // child: list of 3
			0x03, 0x03, 0x04, // child parameter_name
			0x62, 0xFF, // child parameter_value u8(255)
			0x01, // child child_list (absent)
		}
		d := newDecoder(input)
		got := d.readTreeEntry()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if !bytes.Equal(got.ParameterName, []byte{0x01, 0x02}) {
			t.Fatalf("ParameterName = %X, want 0102", got.ParameterName)
		}
		if got.ParameterValue != nil {
			t.Fatalf("ParameterValue = %v, want nil", got.ParameterValue)
		}
		if len(got.Children) != 1 {
			t.Fatalf("len(Children) = %d, want 1", len(got.Children))
		}
		child := got.Children[0]
		if !bytes.Equal(child.ParameterName, []byte{0x03, 0x04}) {
			t.Fatalf("child.ParameterName = %X, want 0304", child.ParameterName)
		}
		if child.ParameterValue != Uint8(255) {
			t.Fatalf("child.ParameterValue = %v (%T), want Uint8(255)", child.ParameterValue, child.ParameterValue)
		}
		if child.Children != nil {
			t.Fatalf("child.Children = %v, want nil", child.Children)
		}
	})
}

func TestDecoderReadOptionalTime(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", got)
		}
	})

	t.Run("present returns time", func(t *testing.T) {
		// Normal 2-element list time
		input := []byte{0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39}
		d := newDecoder(input)
		got := d.readOptionalTime()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil Time")
		}
		if got.Tag != TimeSecIndex {
			t.Fatalf("Tag = %d, want %d", got.Tag, TimeSecIndex)
		}
		if got.Value != 12345 {
			t.Fatalf("Value = %d, want 12345", got.Value)
		}
	})
}

func TestDecoderReadOptionalUnsignedPtr(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalUnsignedPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", *got)
		}
	})

	t.Run("present u8 returns pointer", func(t *testing.T) {
		d := newDecoder([]byte{0x62, 0x2A})
		got := d.readOptionalUnsignedPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil")
		}
		if *got != 42 {
			t.Fatalf("got %d, want 42", *got)
		}
	})

	t.Run("present u32 returns pointer", func(t *testing.T) {
		d := newDecoder([]byte{0x65, 0x00, 0x01, 0x00, 0x00})
		got := d.readOptionalUnsignedPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil")
		}
		if *got != 65536 {
			t.Fatalf("got %d, want 65536", *got)
		}
	})
}

func TestDecoderReadOptionalSignedPtr(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalSignedPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", *got)
		}
	})

	t.Run("present i8 returns pointer", func(t *testing.T) {
		// 0x52 0xFF = Int8(-1)
		d := newDecoder([]byte{0x52, 0xFF})
		got := d.readOptionalSignedPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil")
		}
		if *got != -1 {
			t.Fatalf("got %d, want -1", *got)
		}
	})
}

func TestDecoderReadOptionalStringPtr(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalStringPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", *got)
		}
	})

	t.Run("present returns pointer", func(t *testing.T) {
		// 0x04 0x41 0x42 0x43 = octet string "ABC"
		d := newDecoder([]byte{0x04, 0x41, 0x42, 0x43})
		got := d.readOptionalStringPtr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil")
		}
		if *got != "ABC" {
			t.Fatalf("got %q, want %q", *got, "ABC")
		}
	})
}

func TestDecoderReadOptionalUint8Ptr(t *testing.T) {
	t.Run("skipped returns nil", func(t *testing.T) {
		d := newDecoder([]byte{0x01})
		got := d.readOptionalUint8Ptr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got != nil {
			t.Fatalf("got %v, want nil for skipped optional", *got)
		}
	})

	t.Run("present returns pointer", func(t *testing.T) {
		// 0x62 0x03 = u8(3)
		d := newDecoder([]byte{0x62, 0x03})
		got := d.readOptionalUint8Ptr()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil")
		}
		if *got != 3 {
			t.Fatalf("got %d, want 3", *got)
		}
	})
}
