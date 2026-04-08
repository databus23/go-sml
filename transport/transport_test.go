package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// buildFrame constructs a valid SML transport frame for the given payload.
// It handles padding, escaping, and CRC computation.
func buildFrame(payload []byte) []byte {
	// 1. Escape any 1b1b1b1b sequences in the payload by doubling them.
	escaped := escapePayload(payload)

	// 2. Compute padding to make (escaped payload + padding) divisible by 4.
	padCount := (4 - len(escaped)%4) % 4
	padding := make([]byte, padCount)

	// 3. Build raw frame: start + escaped payload + padding + end escape + 1a + padCount
	var raw []byte
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01) // start
	raw = append(raw, escaped...)                                        // escaped payload
	raw = append(raw, padding...)                                        // padding
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b)                           // end escape
	raw = append(raw, 0x1a)                                              // end marker
	raw = append(raw, byte(padCount))                                    // pad count

	// 4. Compute CRC over raw (start through padCount byte).
	crc := crc16(raw)

	// 5. Append CRC as 2 bytes big-endian.
	var crcBytes [2]byte
	binary.BigEndian.PutUint16(crcBytes[:], crc)
	raw = append(raw, crcBytes[:]...)

	return raw
}

// buildFrameKermit is like buildFrame but uses Kermit CRC.
func buildFrameKermit(payload []byte) []byte {
	escaped := escapePayload(payload)
	padCount := (4 - len(escaped)%4) % 4
	padding := make([]byte, padCount)

	var raw []byte
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01)
	raw = append(raw, escaped...)
	raw = append(raw, padding...)
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b)
	raw = append(raw, 0x1a)
	raw = append(raw, byte(padCount))

	crc := crc16Kermit(raw)
	var crcBytes [2]byte
	binary.BigEndian.PutUint16(crcBytes[:], crc)
	raw = append(raw, crcBytes[:]...)

	return raw
}

// escapePayload doubles any 1b1b1b1b sequences in the data.
func escapePayload(data []byte) []byte {
	marker := []byte{0x1b, 0x1b, 0x1b, 0x1b}
	var result []byte
	i := 0
	for i < len(data) {
		if i+4 <= len(data) && bytes.Equal(data[i:i+4], marker) {
			// Double the escape sequence
			result = append(result, marker...)
			result = append(result, marker...)
			i += 4
		} else {
			result = append(result, data[i])
			i++
		}
	}
	return result
}

func TestReaderSimpleFrame(t *testing.T) {
	payload := []byte{0x76, 0x05, 0x01, 0x02, 0x03, 0x04, 0x62, 0x00}
	frame := buildFrame(payload)
	r := NewReader(bytes.NewReader(frame))

	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Next() = %x, want %x", got, payload)
	}
}

func TestReaderEscapeSequence(t *testing.T) {
	// Payload contains a literal 1b1b1b1b sequence.
	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0x1b, 0x1b, 0x1b, 0x1b, 0x11, 0x22, 0x33, 0x44}
	frame := buildFrame(payload)
	r := NewReader(bytes.NewReader(frame))

	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Next() = %x, want %x", got, payload)
	}
}

func TestReaderPadding(t *testing.T) {
	// Payload that isn't a multiple of 4 bytes, so padding is needed.
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	frame := buildFrame(payload)
	r := NewReader(bytes.NewReader(frame))

	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Next() = %x, want %x", got, payload)
	}
}

func TestReaderMultipleFrames(t *testing.T) {
	payload1 := []byte{0x01, 0x02, 0x03, 0x04}
	payload2 := []byte{0x05, 0x06, 0x07, 0x08}
	var wire []byte
	wire = append(wire, buildFrame(payload1)...)
	wire = append(wire, buildFrame(payload2)...)

	r := NewReader(bytes.NewReader(wire))

	got1, err := r.Next()
	if err != nil {
		t.Fatalf("Next() #1 error = %v", err)
	}
	if !bytes.Equal(got1, payload1) {
		t.Fatalf("Next() #1 = %x, want %x", got1, payload1)
	}

	got2, err := r.Next()
	if err != nil {
		t.Fatalf("Next() #2 error = %v", err)
	}
	if !bytes.Equal(got2, payload2) {
		t.Fatalf("Next() #2 = %x, want %x", got2, payload2)
	}

	// Third call should return EOF.
	_, err = r.Next()
	if err != io.EOF {
		t.Fatalf("Next() #3 error = %v, want io.EOF", err)
	}
}

func TestReaderGarbageBeforeStart(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	frame := buildFrame(payload)
	// Prepend random garbage bytes.
	garbage := []byte{0xFF, 0xFE, 0x00, 0x42, 0x1b, 0x1b, 0x00, 0x13, 0x37}
	var wire []byte
	wire = append(wire, garbage...)
	wire = append(wire, frame...)

	r := NewReader(bytes.NewReader(wire))

	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Next() = %x, want %x", got, payload)
	}
}

func TestReaderCRCFailure(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	frame := buildFrame(payload)
	// Corrupt the last byte (part of CRC).
	frame[len(frame)-1] ^= 0xFF

	r := NewReader(bytes.NewReader(frame))

	_, err := r.Next()
	if err == nil {
		t.Fatal("Next() expected CRC error, got nil")
	}
}

func TestReaderKermitCRC(t *testing.T) {
	payload := []byte{0x76, 0x05, 0x01, 0x02, 0x03, 0x04, 0x62, 0x00}
	frame := buildFrameKermit(payload)

	r := NewReader(bytes.NewReader(frame))

	got, err := r.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Next() = %x, want %x", got, payload)
	}
}

func TestReaderMaxFrameSize(t *testing.T) {
	// Create a payload larger than max frame size (64KB).
	payload := make([]byte, 65*1024)
	for i := range payload {
		payload[i] = byte(i)
	}
	// Ensure no accidental 1b1b1b1b sequences to keep the test simple.
	// The payload is large enough that we exceed the limit regardless.
	frame := buildFrame(payload)

	r := NewReader(bytes.NewReader(frame))

	_, err := r.Next()
	if err == nil {
		t.Fatal("Next() expected max frame size error, got nil")
	}
}

func TestReaderEOF(t *testing.T) {
	r := NewReader(bytes.NewReader(nil))

	_, err := r.Next()
	if err != io.EOF {
		t.Fatalf("Next() error = %v, want io.EOF", err)
	}
}
