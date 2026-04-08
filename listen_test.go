package sml

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// crcTable is copied from the transport package (unexported there).
var testCRCTable = func() [256]uint16 {
	var t [256]uint16
	for i := range t {
		crc := uint16(i)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
		t[i] = crc
	}
	return t
}()

// testCRC16 computes CRC-16/X.25 with the DIN 62056-46 byte-swap.
func testCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc = (crc >> 8) ^ testCRCTable[(crc^uint16(b))&0xFF]
	}
	crc ^= 0xFFFF
	return (crc << 8) | (crc >> 8)
}

// buildTestFrame wraps payload bytes in a valid SML transport frame.
func buildTestFrame(payload []byte) []byte {
	// Escape any 1b1b1b1b sequences in payload.
	marker := []byte{0x1b, 0x1b, 0x1b, 0x1b}
	var escaped []byte
	i := 0
	for i < len(payload) {
		if i+4 <= len(payload) && bytes.Equal(payload[i:i+4], marker) {
			escaped = append(escaped, marker...)
			escaped = append(escaped, marker...)
			i += 4
		} else {
			escaped = append(escaped, payload[i])
			i++
		}
	}

	padCount := (4 - len(escaped)%4) % 4
	padding := make([]byte, padCount)

	var raw []byte
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01) // start
	raw = append(raw, escaped...)
	raw = append(raw, padding...)
	raw = append(raw, 0x1b, 0x1b, 0x1b, 0x1b) // end escape
	raw = append(raw, 0x1a)
	raw = append(raw, byte(padCount))

	crc := testCRC16(raw)
	var crcBytes [2]byte
	binary.BigEndian.PutUint16(crcBytes[:], crc)
	raw = append(raw, crcBytes[:]...)

	return raw
}

// buildValidSMLPayload builds a minimal valid SML file payload (Open + Close).
func buildValidSMLPayload() []byte {
	var buf []byte
	buf = append(buf, buildOpenResponseMsg(0x01)...)
	buf = append(buf, buildCloseResponseMsg(0x02)...)
	return buf
}

func TestListen(t *testing.T) {
	payload1 := buildValidSMLPayload()
	payload2 := buildValidSMLPayload()

	var wire []byte
	wire = append(wire, buildTestFrame(payload1)...)
	wire = append(wire, buildTestFrame(payload2)...)

	var called int
	handler := func(f *File) error {
		called++
		return nil
	}

	err := Listen(context.Background(), bytes.NewReader(wire), handler)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Listen() error = %v, want nil or EOF", err)
	}
	if called != 2 {
		t.Fatalf("handler called %d times, want 2", called)
	}
}

func TestListenHandlerStop(t *testing.T) {
	payload := buildValidSMLPayload()

	var wire []byte
	wire = append(wire, buildTestFrame(payload)...)
	wire = append(wire, buildTestFrame(payload)...)

	sentinel := errors.New("stop now")
	var called int
	handler := func(f *File) error {
		called++
		return sentinel
	}

	err := Listen(context.Background(), bytes.NewReader(wire), handler)
	if !errors.Is(err, sentinel) {
		t.Fatalf("Listen() error = %v, want sentinel error", err)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
}

func TestListenContextCancel(t *testing.T) {
	// Use a pipe so the reader blocks after the first frame.
	pr, pw := io.Pipe()

	payload := buildValidSMLPayload()
	frame := buildTestFrame(payload)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		handler := func(f *File) error {
			// Cancel context after first successful call.
			cancel()
			return nil
		}
		errCh <- Listen(ctx, pr, handler)
	}()

	// Write the first frame; Listen will process it and the handler will cancel.
	pw.Write(frame)

	err := <-errCh
	pw.Close()

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Listen() error = %v, want context.Canceled", err)
	}
}

func TestListenSkipsCorruptFrames(t *testing.T) {
	payload := buildValidSMLPayload()
	goodFrame := buildTestFrame(payload)

	// Corrupt frame: valid start sequence, garbage payload, bad CRC.
	corruptFrame := []byte{
		0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01, // start
		0xDE, 0xAD, 0xBE, 0xEF, // garbage payload (4 bytes, no padding needed)
		0x1b, 0x1b, 0x1b, 0x1b, 0x1a, 0x00, // end escape + end marker + pad count=0
		0xFF, 0xFF, // deliberately wrong CRC
	}

	var wire []byte
	wire = append(wire, goodFrame...)
	wire = append(wire, corruptFrame...)
	wire = append(wire, goodFrame...)

	var called int
	handler := func(f *File) error {
		called++
		return nil
	}

	err := Listen(context.Background(), bytes.NewReader(wire), handler)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Listen() error = %v, want nil or EOF", err)
	}
	if called != 2 {
		t.Fatalf("handler called %d times, want 2 (corrupt frame should be skipped)", called)
	}
}

func TestListenMessages(t *testing.T) {
	// Payload has 2 messages: OpenResponse + CloseResponse.
	payload := buildValidSMLPayload()
	frame := buildTestFrame(payload)

	var messages []*Message
	handler := func(m *Message) error {
		messages = append(messages, m)
		return nil
	}

	err := ListenMessages(context.Background(), bytes.NewReader(frame), handler)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("ListenMessages() error = %v, want nil or EOF", err)
	}
	if len(messages) != 2 {
		t.Fatalf("handler called %d times, want 2", len(messages))
	}
	if _, ok := messages[0].Body.(*OpenResponse); !ok {
		t.Fatalf("messages[0].Body is %T, want *OpenResponse", messages[0].Body)
	}
	if _, ok := messages[1].Body.(*CloseResponse); !ok {
		t.Fatalf("messages[1].Body is %T, want *CloseResponse", messages[1].Body)
	}
}
