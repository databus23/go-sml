package sml

import (
	"context"
	"errors"
	"io"

	"github.com/databus23/go-sml/transport"
)

// Handler is called for each decoded SML file. Return non-nil error to stop listening.
type Handler func(*File) error

// MessageHandler is called for each message. Return non-nil error to stop listening.
type MessageHandler func(*Message) error

// Listen reads SML frames from r, decodes them, and calls handler for each file.
// Blocks until r is exhausted (io.EOF), ctx is cancelled, or handler returns a non-nil error.
// Frames that fail CRC validation or SML decoding are silently skipped.
func Listen(ctx context.Context, r io.Reader, handler Handler) error {
	reader := transport.NewReader(r)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		data, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			// Non-fatal transport error (e.g. CRC mismatch): skip and continue.
			continue
		}

		file, err := Decode(data)
		if err != nil {
			// Malformed SML payload: skip and continue.
			continue
		}

		if err := handler(file); err != nil {
			return err
		}
	}
}

// ListenMessages calls handler once per message rather than once per file.
// Frames that fail CRC validation or SML decoding are silently skipped.
func ListenMessages(ctx context.Context, r io.Reader, handler MessageHandler) error {
	return Listen(ctx, r, func(f *File) error {
		for _, msg := range f.Messages {
			if err := handler(msg); err != nil {
				return err
			}
		}
		return nil
	})
}
