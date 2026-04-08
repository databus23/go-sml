package sml

import "io"

// decoder reads SML TLV-encoded data from a byte buffer.
type decoder struct {
	buf []byte
	pos int
	err error // sticky: once set, all methods are no-ops
}

func newDecoder(buf []byte) *decoder {
	return &decoder{buf: buf}
}

// readByte reads a single byte and advances the position.
// Returns 0 if an error is already set or the buffer is exhausted.
func (d *decoder) readByte() byte {
	if d.err != nil {
		return 0
	}
	if d.pos >= len(d.buf) {
		d.err = io.ErrUnexpectedEOF
		return 0
	}
	b := d.buf[d.pos]
	d.pos++
	return b
}

// readTypeLength reads a TLV type-length header and returns the type and
// data length. For non-list types, the returned length is the number of data
// bytes (total length minus TL byte count). For list types, the returned
// length is the element count.
func (d *decoder) readTypeLength() (typ byte, length int) {
	if d.err != nil {
		return 0, 0
	}

	b := d.readByte()
	if d.err != nil {
		return 0, 0
	}

	typ = b & 0x70
	length = int(b & 0x0F)
	tlBytes := 1

	for b&0x80 != 0 {
		b = d.readByte()
		if d.err != nil {
			return 0, 0
		}
		length = (length << 4) | int(b&0x0F)
		tlBytes++
	}

	if typ != 0x70 { // not a list
		length -= tlBytes
	}

	return typ, length
}

// isEndOfMessage peeks at the current byte and returns true if it is 0x00
// (end-of-message marker). Does not consume the byte.
func (d *decoder) isEndOfMessage() bool {
	if d.err != nil {
		return false
	}
	if d.pos >= len(d.buf) {
		return false
	}
	return d.buf[d.pos] == 0x00
}

// isOptionalSkipped checks if the current byte is 0x01 (optional field absent).
// If so, it consumes the byte and returns true.
func (d *decoder) isOptionalSkipped() bool {
	if d.err != nil {
		return false
	}
	if d.pos >= len(d.buf) {
		return false
	}
	if d.buf[d.pos] == 0x01 {
		d.pos++
		return true
	}
	return false
}
