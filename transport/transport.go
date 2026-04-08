package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const maxFrameSize = 64 * 1024

var (
	startSequence = [8]byte{0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01}
	escapeMarker  = [4]byte{0x1b, 0x1b, 0x1b, 0x1b}
)

// Reader reads SML frames from an io.Reader, handling transport layer
// framing, escape sequences, CRC validation, and padding removal.
type Reader struct {
	r   io.Reader
	buf [1]byte // single-byte read buffer
}

// NewReader returns a Reader that reads SML transport frames from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Next returns the next complete SML frame payload (transport stripped, CRC validated).
// Returns io.EOF when the underlying reader is exhausted.
func (r *Reader) Next() ([]byte, error) {
	if err := r.scanForStart(); err != nil {
		return nil, err
	}

	// rawFrame accumulates everything from the start sequence onward (for CRC).
	rawFrame := make([]byte, 0, 256)
	rawFrame = append(rawFrame, startSequence[:]...)

	// unescaped accumulates the payload with escape sequences resolved.
	var unescaped []byte

	// Read 4 bytes at a time after the start sequence.
	var quad [4]byte
	for {
		if err := r.readFull(quad[:]); err != nil {
			return nil, err
		}
		rawFrame = append(rawFrame, quad[:]...)

		if len(rawFrame) > maxFrameSize {
			return nil, fmt.Errorf("transport: frame exceeds maximum size of %d bytes", maxFrameSize)
		}

		if quad != escapeMarker {
			// Normal data bytes.
			unescaped = append(unescaped, quad[:]...)
			continue
		}

		// We saw 1b1b1b1b -- peek at the next 4 bytes to decide what it means.
		var next [4]byte
		if err := r.readFull(next[:]); err != nil {
			return nil, err
		}
		rawFrame = append(rawFrame, next[:]...)

		if next == escapeMarker {
			// Escape sequence: doubled 1b1b1b1b → emit one literal 1b1b1b1b.
			unescaped = append(unescaped, escapeMarker[:]...)
			continue
		}

		if next[0] == 0x1a {
			// End of frame. next = [1a, padCount, crcHi, crcLo]
			padCount := int(next[1])
			wireCRC := binary.BigEndian.Uint16(next[2:4])

			// CRC is computed over everything from start through padCount byte
			// (excluding the 2 CRC bytes themselves).
			crcData := rawFrame[:len(rawFrame)-2]
			computedCRC := crc16(crcData)
			computedKermit := crc16Kermit(crcData)

			if wireCRC != computedCRC && wireCRC != computedKermit {
				return nil, fmt.Errorf("transport: CRC mismatch: wire 0x%04x, computed X.25 0x%04x, Kermit 0x%04x",
					wireCRC, computedCRC, computedKermit)
			}

			// Strip padding from unescaped payload.
			if padCount > 0 {
				if padCount > len(unescaped) {
					return nil, fmt.Errorf("transport: pad count %d exceeds payload length %d", padCount, len(unescaped))
				}
				unescaped = unescaped[:len(unescaped)-padCount]
			}

			return unescaped, nil
		}

		return nil, fmt.Errorf("transport: unexpected byte 0x%02x after escape marker", next[0])
	}
}

// scanForStart scans the stream for the 8-byte start sequence.
// Returns io.EOF if the stream ends before a start sequence is found.
func (r *Reader) scanForStart() error {
	// We track how many consecutive bytes of the start sequence we've matched.
	matched := 0
	for matched < 8 {
		b, err := r.readByte()
		if err != nil {
			return err
		}
		if b == startSequence[matched] {
			matched++
		} else if b == startSequence[0] {
			// The byte could be the start of a new match attempt.
			// Since startSequence starts with 4x 0x1b then 4x 0x01,
			// if we see 0x1b while partially matched, restart from position 1.
			matched = 1
		} else {
			matched = 0
		}
	}
	return nil
}

// readByte reads a single byte from the underlying reader.
func (r *Reader) readByte() (byte, error) {
	_, err := io.ReadFull(r.r, r.buf[:])
	return r.buf[0], err
}

// readFull reads exactly len(p) bytes from the underlying reader.
func (r *Reader) readFull(p []byte) error {
	_, err := io.ReadFull(r.r, p)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
