package sml_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databus23/go-sml"
	"github.com/databus23/go-sml/transport"
)

// collectSMLServerOutput reads all frames from a .bin file, decodes them,
// and formats all GetListResponse entries matching C sml_server output.
func collectSMLServerOutput(binData []byte) []string {
	reader := transport.NewReader(bytes.NewReader(binData))
	var lines []string

	for {
		frame, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip bad frames (transport errors), keep trying.
			continue
		}

		// DecodeWithOptions with Strict=false always returns a non-nil file,
		// collecting partial results alongside errors.
		file, _ := sml.DecodeWithOptions(frame, sml.DecodeOptions{Strict: false})

		for _, msg := range file.Messages {
			glr, ok := msg.Body.(*sml.GetListResponse)
			if !ok {
				continue
			}
			for i := range glr.ValList {
				line, ok := sml.FormatReadingValue(&glr.ValList[i])
				if ok {
					lines = append(lines, line)
				}
			}
		}
	}

	return lines
}

// filterGoldenLines removes C sml_server debug/error lines from golden file content.
// These are lines that start with "libsml:" or contain "Error in data stream".
func filterGoldenLines(raw string) []string {
	rawLines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	var filtered []string
	for _, line := range rawLines {
		if strings.HasPrefix(line, "libsml:") {
			continue
		}
		if strings.Contains(line, "Error in data stream") {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func TestGoldenFiles(t *testing.T) {
	binFiles, err := filepath.Glob("testdata/*.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(binFiles) == 0 {
		t.Fatal("no .bin files found in testdata/")
	}

	for _, binPath := range binFiles {
		baseName := strings.TrimSuffix(filepath.Base(binPath), ".bin")
		t.Run(baseName, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "golden", baseName+".txt")
			goldenData, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden file: %v", err)
			}
			goldenContent := string(goldenData)

			// Check if this is an error-only golden file.
			isErrorGolden := strings.HasPrefix(goldenContent, "libsml: error:")
			goldenLines := filterGoldenLines(goldenContent)

			binData, err := os.ReadFile(binPath)
			if err != nil {
				t.Fatalf("reading bin file: %v", err)
			}

			gotLines := collectSMLServerOutput(binData)

			if isErrorGolden && len(goldenLines) == 0 {
				// Error golden with no data lines — verify parser produces no output.
				if len(gotLines) > 0 {
					t.Errorf("expected no output for error golden, got %d lines", len(gotLines))
				}
				return
			}

			if len(gotLines) != len(goldenLines) {
				t.Errorf("line count mismatch: got %d, want %d", len(gotLines), len(goldenLines))
				// Show first divergence for debugging.
				minLen := min(len(gotLines), len(goldenLines))
				for i := 0; i < minLen; i++ {
					if gotLines[i] != goldenLines[i] {
						t.Errorf("first mismatch at line %d:\n  got:  %q\n  want: %q", i+1, gotLines[i], goldenLines[i])
						break
					}
				}
				if len(gotLines) > minLen {
					t.Errorf("extra got line %d: %q", minLen+1, gotLines[minLen])
				}
				if len(goldenLines) > minLen {
					t.Errorf("missing golden line %d: %q", minLen+1, goldenLines[minLen])
				}
				return
			}

			for i := range goldenLines {
				if gotLines[i] != goldenLines[i] {
					t.Errorf("line %d mismatch:\n  got:  %q\n  want: %q", i+1, gotLines[i], goldenLines[i])
				}
			}
		})
	}
}
