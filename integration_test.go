package sml_test

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databus23/go-sml"
	"github.com/databus23/go-sml/transport"
)

// formatOctetStringMixed formats an OctetString using the C sml_server "mixed"
// algorithm: bytes in range (0x20, 0x7b) are output as ASCII; once any byte falls
// outside that range, all remaining bytes use lowercase hex with trailing space.
// The thresholds match sml_server exactly (not general ASCII printability).
func formatOctetStringMixed(data []byte) string {
	var b strings.Builder
	mixed := true
	for _, c := range data {
		if mixed && c > 0x20 && c < 0x7b {
			b.WriteByte(c)
		} else {
			mixed = false
			fmt.Fprintf(&b, "%02x ", c)
		}
	}
	return b.String()
}

// formatNumericValue formats an integer or unsigned value with scaler applied,
// matching the C sml_server precision rules.
func formatNumericValue(raw int64, scaler int8) string {
	precision := 0
	if scaler < 0 {
		precision = int(-scaler)
	}
	scaled := float64(raw) * math.Pow(10, float64(scaler))
	return fmt.Sprintf("%.*f", precision, scaled)
}

// rawInt64 extracts the raw integer value from a numeric SML value type.
// Returns (value, true) for numeric types, (0, false) otherwise.
func rawInt64(v sml.Value) (int64, bool) {
	switch val := v.(type) {
	case sml.Int8:
		return int64(val), true
	case sml.Int16:
		return int64(val), true
	case sml.Int32:
		return int64(val), true
	case sml.Int64:
		return int64(val), true
	case sml.Uint8:
		return int64(val), true
	case sml.Uint16:
		return int64(val), true
	case sml.Uint32:
		return int64(val), true
	case sml.Uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

// formatListEntry formats a single ListEntry matching C sml_server output.
// Returns ("", false) if the entry should be skipped (nil value).
func formatListEntry(entry *sml.ListEntry) (string, bool) {
	if entry.Value == nil {
		return "", false
	}

	obis := entry.OBISString()
	if obis == "" {
		return "", false
	}

	var valuePart string
	switch val := entry.Value.(type) {
	case sml.OctetString:
		valuePart = formatOctetStringMixed([]byte(val))
	case sml.Bool:
		if bool(val) {
			valuePart = "true"
		} else {
			valuePart = "false"
		}
	default:
		raw, ok := rawInt64(entry.Value)
		if !ok {
			return "", false
		}
		var scaler int8
		if entry.Scaler != nil {
			scaler = *entry.Scaler
		}
		valuePart = formatNumericValue(raw, scaler)
	}

	unitPart := ""
	if entry.Unit != nil {
		unitPart = sml.UnitName(*entry.Unit)
	}

	return fmt.Sprintf("%s#%s#%s", obis, valuePart, unitPart), true
}

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
				line, ok := formatListEntry(&glr.ValList[i])
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
