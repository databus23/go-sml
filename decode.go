package sml

import (
	"fmt"
	"io"
)

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

// readBytes reads n bytes from the buffer and returns them as a new slice (copy).
func (d *decoder) readBytes(n int) []byte {
	if d.err != nil {
		return nil
	}
	if d.pos+n > len(d.buf) {
		d.err = io.ErrUnexpectedEOF
		return nil
	}
	out := make([]byte, n)
	copy(out, d.buf[d.pos:d.pos+n])
	d.pos += n
	return out
}

// readOctetString reads a TLV-encoded octet string. The returned slice is a
// copy of the data, not a reference into the input buffer.
func (d *decoder) readOctetString() []byte {
	typ, length := d.readTypeLength()
	if d.err != nil {
		return nil
	}
	if typ != 0x00 {
		d.err = fmt.Errorf("sml: expected octet string (type 0x00), got 0x%02X", typ)
		return nil
	}
	return d.readBytes(length)
}

// readBool reads a TLV-encoded boolean value.
func (d *decoder) readBool() bool {
	typ, length := d.readTypeLength()
	if d.err != nil {
		return false
	}
	if typ != 0x40 {
		d.err = fmt.Errorf("sml: expected boolean (type 0x40), got 0x%02X", typ)
		return false
	}
	if length != 1 {
		d.err = fmt.Errorf("sml: boolean data length = %d, want 1", length)
		return false
	}
	return d.readByte() != 0
}

// readUnsigned reads a TLV-encoded unsigned integer and returns the
// appropriately sized concrete Value type (Uint8, Uint16, Uint32, or Uint64).
// Data lengths that are not a power of 2 are rounded up (e.g., 3 bytes → Uint32).
func (d *decoder) readUnsigned() Value {
	typ, length := d.readTypeLength()
	if d.err != nil {
		return nil
	}
	if typ != 0x60 {
		d.err = fmt.Errorf("sml: expected unsigned (type 0x60), got 0x%02X", typ)
		return nil
	}
	data := d.readBytes(length)
	if d.err != nil {
		return nil
	}
	// Accumulate big-endian value
	var val uint64
	for _, b := range data {
		val = (val << 8) | uint64(b)
	}
	switch {
	case length <= 1:
		return Uint8(val)
	case length <= 2:
		return Uint16(val)
	case length <= 4:
		return Uint32(val)
	default:
		return Uint64(val)
	}
}

// readSigned reads a TLV-encoded signed integer and returns the appropriately
// sized concrete Value type (Int8, Int16, Int32, or Int64). Sign extension is
// applied when the data length is not a power of 2.
func (d *decoder) readSigned() Value {
	typ, length := d.readTypeLength()
	if d.err != nil {
		return nil
	}
	if typ != 0x50 {
		d.err = fmt.Errorf("sml: expected signed (type 0x50), got 0x%02X", typ)
		return nil
	}
	data := d.readBytes(length)
	if d.err != nil {
		return nil
	}
	// Accumulate big-endian with sign extension: if high bit of first byte
	// is set, pre-fill with 0xFF bytes to sign-extend.
	var val uint64
	if len(data) > 0 && data[0]&0x80 != 0 {
		val = ^uint64(0) // all 1s
	}
	for _, b := range data {
		val = (val << 8) | uint64(b)
	}
	switch {
	case length <= 1:
		return Int8(int8(val))
	case length <= 2:
		return Int16(int16(val))
	case length <= 4:
		return Int32(int32(val))
	default:
		return Int64(int64(val))
	}
}

// readValue peeks at the type of the next TLV element and dispatches to the
// appropriate typed reader. Returns a concrete Value (OctetString, Bool,
// Uint8..Uint64, Int8..Int64).
func (d *decoder) readValue() Value {
	if d.err != nil {
		return nil
	}
	if d.pos >= len(d.buf) {
		d.err = io.ErrUnexpectedEOF
		return nil
	}
	// Peek at type bits without consuming the byte
	typ := d.buf[d.pos] & 0x70
	switch typ {
	case 0x00:
		return OctetString(d.readOctetString())
	case 0x40:
		return Bool(d.readBool())
	case 0x50:
		return d.readSigned()
	case 0x60:
		return d.readUnsigned()
	default:
		d.err = fmt.Errorf("sml: unexpected type 0x%02X in readValue", typ)
		return nil
	}
}

// readListLength reads a TLV list header and returns the element count.
func (d *decoder) readListLength() int {
	typ, length := d.readTypeLength()
	if d.err != nil {
		return 0
	}
	if typ != 0x70 {
		d.err = fmt.Errorf("sml: expected list (type 0x70), got 0x%02X", typ)
		return 0
	}
	return length
}

// readOptionalOctetString reads an optional octet string. Returns nil if the
// field is absent (0x01 marker).
func (d *decoder) readOptionalOctetString() []byte {
	if d.isOptionalSkipped() {
		return nil
	}
	return d.readOctetString()
}

// readOptionalValue reads an optional SML value. Returns nil if the field is
// absent (0x01 marker).
func (d *decoder) readOptionalValue() Value {
	if d.isOptionalSkipped() {
		return nil
	}
	return d.readValue()
}

// readTime reads an SML Time structure. The standard encoding is a 2-element
// list (tag + value). As a workaround for Holley DTZ541 meters, a bare unsigned
// integer (no list wrapper) is also accepted and treated as TimeSecIndex.
func (d *decoder) readTime() Time {
	if d.err != nil {
		return Time{}
	}
	// Peek at the current byte to determine encoding variant.
	if d.pos >= len(d.buf) {
		d.err = io.ErrUnexpectedEOF
		return Time{}
	}
	typ := d.buf[d.pos] & 0x70
	if typ == 0x70 {
		// Standard: 2-element list
		_ = d.readListLength()
		tag := d.readUnsigned()
		val := d.readUnsigned()
		if d.err != nil {
			return Time{}
		}
		return Time{Tag: uint8(tag.(Uint8)), Value: uint32(toUint64(val))}
	}
	// Holley workaround: bare unsigned
	val := d.readUnsigned()
	if d.err != nil {
		return Time{}
	}
	return Time{Tag: TimeSecIndex, Value: uint32(toUint64(val))}
}

// readOptionalTime reads an optional Time. Returns nil if skipped.
func (d *decoder) readOptionalTime() *Time {
	if d.isOptionalSkipped() {
		return nil
	}
	t := d.readTime()
	if d.err != nil {
		return nil
	}
	return &t
}

// readTreePath reads a list of octet strings representing a tree path.
func (d *decoder) readTreePath() TreePath {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	path := make(TreePath, n)
	for i := range path {
		path[i] = d.readOctetString()
	}
	return path
}

// readTreeEntry reads a recursive 3-element tree entry (parameter_name,
// parameter_value, child_list).
func (d *decoder) readTreeEntry() TreeEntry {
	_ = d.readListLength()
	name := d.readOctetString()
	value := d.readOptionalValue()
	var children []TreeEntry
	if !d.isOptionalSkipped() {
		n := d.readListLength()
		if d.err != nil {
			return TreeEntry{}
		}
		children = make([]TreeEntry, n)
		for i := range children {
			children[i] = d.readTreeEntry()
		}
	}
	if d.err != nil {
		return TreeEntry{}
	}
	return TreeEntry{
		ParameterName:  name,
		ParameterValue: value,
		Children:       children,
	}
}

// readOptionalUnsignedPtr reads an optional unsigned integer and returns a
// *uint64. Returns nil if the field is absent.
func (d *decoder) readOptionalUnsignedPtr() *uint64 {
	if d.isOptionalSkipped() {
		return nil
	}
	v := d.readUnsigned()
	if d.err != nil {
		return nil
	}
	u := toUint64(v)
	return &u
}

// readOptionalSignedPtr reads an optional signed integer and returns a *int8.
// Returns nil if the field is absent.
func (d *decoder) readOptionalSignedPtr() *int8 {
	if d.isOptionalSkipped() {
		return nil
	}
	v := d.readSigned()
	if d.err != nil {
		return nil
	}
	s := toInt8(v)
	return &s
}

// readOptionalStringPtr reads an optional octet string and returns it as a
// *string. Returns nil if the field is absent.
func (d *decoder) readOptionalStringPtr() *string {
	if d.isOptionalSkipped() {
		return nil
	}
	b := d.readOctetString()
	if d.err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// readOptionalUint8Ptr reads an optional unsigned integer and returns a *uint8.
// Returns nil if the field is absent.
func (d *decoder) readOptionalUint8Ptr() *uint8 {
	if d.isOptionalSkipped() {
		return nil
	}
	v := d.readUnsigned()
	if d.err != nil {
		return nil
	}
	u := uint8(toUint64(v))
	return &u
}

// toUint64 converts any concrete unsigned Value to uint64.
func toUint64(v Value) uint64 {
	switch v := v.(type) {
	case Uint8:
		return uint64(v)
	case Uint16:
		return uint64(v)
	case Uint32:
		return uint64(v)
	case Uint64:
		return uint64(v)
	default:
		return 0
	}
}

// toInt8 converts any concrete signed Value to int8.
func toInt8(v Value) int8 {
	switch v := v.(type) {
	case Int8:
		return int8(v)
	case Int16:
		return int8(v)
	case Int32:
		return int8(v)
	case Int64:
		return int8(v)
	default:
		return 0
	}
}
