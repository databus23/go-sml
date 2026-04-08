package sml

import (
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
