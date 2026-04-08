package sml_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databus23/go-sml"
	"github.com/databus23/go-sml/transport"
)

// FuzzDecode fuzzes sml.Decode with arbitrary byte slices.
// Seeds are extracted SML payloads (transport-stripped frames) from testdata/*.bin files.
func FuzzDecode(f *testing.F) {
	binFiles, err := filepath.Glob("testdata/*.bin")
	if err != nil {
		f.Fatal(err)
	}

	for _, binPath := range binFiles {
		data, err := os.ReadFile(binPath)
		if err != nil {
			f.Fatalf("reading %s: %v", binPath, err)
		}
		reader := transport.NewReader(bytes.NewReader(data))
		for {
			frame, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			f.Add(frame)
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic; errors are expected and fine.
		sml.Decode(data) //nolint:errcheck
	})
}

// FuzzTransportReader fuzzes transport.NewReader(...).Next() with arbitrary raw
// transport data. Seeds are the raw testdata/*.bin file contents.
func FuzzTransportReader(f *testing.F) {
	binFiles, err := filepath.Glob("testdata/*.bin")
	if err != nil {
		f.Fatal(err)
	}

	for _, binPath := range binFiles {
		data, err := os.ReadFile(binPath)
		if err != nil {
			f.Fatalf("reading %s: %v", binPath, err)
		}
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic; errors are expected and fine.
		reader := transport.NewReader(bytes.NewReader(data))
		for {
			_, err := reader.Next()
			if err != nil {
				break
			}
		}
	})
}
