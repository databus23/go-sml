package sml

import (
	"errors"
	"fmt"
)

// Decode parses SML binary data in strict mode, returning the first error encountered.
func Decode(data []byte) (*File, error) {
	return DecodeWithOptions(data, DecodeOptions{Strict: true})
}

// DecodeWithOptions parses SML binary data with configurable strictness.
// In strict mode (default), the first parse error aborts decoding.
// In non-strict mode, malformed messages are skipped and partial results
// are returned alongside a joined error.
func DecodeWithOptions(data []byte, opts DecodeOptions) (*File, error) {
	d := newDecoder(data)
	file := &File{}
	var errs []error

	for {
		// Skip trailing 0x00 bytes (end-of-message padding between messages).
		for d.pos < len(d.buf) && d.buf[d.pos] == 0x00 {
			d.pos++
		}
		if d.pos >= len(d.buf) {
			break
		}

		startPos := d.pos
		msg, err := decodeMessage(d)
		if err != nil {
			if opts.Strict {
				return nil, err
			}
			errs = append(errs, err)
			// In non-strict mode, try to recover by scanning forward from
			// one byte past the failed message start, looking for the next
			// 0x76 byte (list-of-6 header) that could begin a new message.
			// We reset to startPos+1 because the decoder may have consumed
			// bytes from subsequent valid messages during the failed parse.
			d.err = nil
			d.pos = startPos + 1
			found := false
			for d.pos < len(d.buf) {
				if d.buf[d.pos] == 0x76 {
					found = true
					break
				}
				d.pos++
			}
			if !found {
				break
			}
			continue
		}
		if msg != nil {
			file.Messages = append(file.Messages, msg)
		}
	}

	if len(errs) > 0 {
		return file, errors.Join(errs...)
	}
	return file, nil
}

// decodeMessage reads a single SML message (6-element list) from the decoder.
// Returns (nil, nil) if the message body tag is not a recognized response (e.g., request tags).
// Returns (nil, error) if parsing fails.
func decodeMessage(d *decoder) (*Message, error) {
	startPos := d.pos
	d.err = nil

	n := d.readListLength()
	if d.err != nil {
		return nil, fmt.Errorf("sml: reading message list header at offset %d: %w", startPos, d.err)
	}
	if n != 6 {
		return nil, fmt.Errorf("sml: message list length = %d at offset %d, want 6", n, startPos)
	}

	txID := d.readOctetString()
	groupID := d.readUnsigned()
	abortOnError := d.readUnsigned()
	if d.err != nil {
		return nil, fmt.Errorf("sml: reading message header at offset %d: %w", startPos, d.err)
	}

	body, err := decodeMessageBody(d)
	if d.err != nil {
		return nil, fmt.Errorf("sml: reading message body at offset %d: %w", startPos, d.err)
	}
	if err != nil {
		return nil, fmt.Errorf("sml: reading message body at offset %d: %w", startPos, err)
	}

	crc := d.readUnsigned()
	// Read end_of_msg marker (0x00)
	endMarker := d.readByte()
	if d.err != nil {
		return nil, fmt.Errorf("sml: reading message trailer at offset %d: %w", startPos, d.err)
	}
	if endMarker != 0x00 {
		return nil, fmt.Errorf("sml: expected end-of-message 0x00 at offset %d, got 0x%02X", d.pos-1, endMarker)
	}

	// If body is nil, the tag was unrecognized (e.g., a request) — skip this message.
	if body == nil {
		return nil, nil
	}

	msg := &Message{
		TransactionID: txID,
		GroupID:       uint8(toUint64(groupID)),
		AbortOnError:  uint8(toUint64(abortOnError)),
		Body:          body,
		CRC:           uint16(toUint64(crc)),
	}
	return msg, nil
}

// decodeMessageBody reads the 2-element list (tag + data) and dispatches to
// the appropriate message parser based on the tag.
// Returns (nil, nil) for unrecognized tags (requests or unimplemented responses).
func decodeMessageBody(d *decoder) (MessageBody, error) {
	n := d.readListLength()
	if d.err != nil {
		return nil, d.err
	}
	if n != 2 {
		return nil, fmt.Errorf("sml: message body list length = %d, want 2", n)
	}

	tagVal := d.readUnsigned()
	if d.err != nil {
		return nil, d.err
	}
	tag := uint32(toUint64(tagVal))

	switch tag {
	case tagOpenResponse:
		return d.readOpenResponse(), d.err
	case tagCloseResponse:
		return d.readCloseResponse(), d.err
	case tagGetListResponse:
		return d.readGetListResponse(), d.err
	case tagAttentionResponse:
		return d.readAttentionResponse(), d.err
	case tagGetProcParameterResponse:
		return d.readGetProcParameterResponse(), d.err
	case tagGetProfileListResponse:
		return d.readGetProfileListResponse(), d.err
	case tagGetProfilePackResponse:
		return d.readGetProfilePackResponse(), d.err
	default:
		// Unknown or request tag — skip past the body data.
		// The body is the next TLV element; skip it by reading its type-length
		// and advancing past the data.
		skipTLV(d)
		if d.err != nil {
			return nil, d.err
		}
		return nil, nil
	}
}

// skipTLV reads and discards one complete TLV element from the decoder.
// For lists, it recursively skips each child element.
func skipTLV(d *decoder) {
	if d.err != nil {
		return
	}
	if d.pos >= len(d.buf) {
		return
	}

	typ, length := d.readTypeLength()
	if d.err != nil {
		return
	}

	if typ == 0x70 {
		// List: skip each child element; cap to remaining bytes to prevent huge loops.
		length = d.safeListLength(length)
		for i := 0; i < length; i++ {
			skipTLV(d)
		}
	} else {
		// Non-list: skip the data bytes
		if d.pos+length > len(d.buf) {
			d.err = fmt.Errorf("sml: skipTLV: need %d bytes at offset %d, have %d", length, d.pos, len(d.buf)-d.pos)
			return
		}
		d.pos += length
	}
}
