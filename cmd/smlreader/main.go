// Command smlreader reads SML binary data and prints meter readings in the
// same format as the C sml_server reference tool (OBIS#value#unit).
//
// Usage:
//
//	smlreader <file>    # read from file
//	smlreader -         # read from stdin
package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/databus23/go-sml"
	"github.com/databus23/go-sml/transport"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <file|->\n", os.Args[0])
		os.Exit(1)
	}

	var r io.Reader
	if os.Args[1] == "-" {
		r = os.Stdin
	} else {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		r = bytes.NewReader(data)
	}

	reader := transport.NewReader(r)
	for {
		frame, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		file, _ := sml.DecodeWithOptions(frame, sml.DecodeOptions{Strict: false})
		for _, msg := range file.Messages {
			glr, ok := msg.Body.(*sml.GetListResponse)
			if !ok {
				continue
			}
			for i := range glr.ValList {
				line, ok := formatListEntry(&glr.ValList[i])
				if ok {
					fmt.Println(line)
				}
			}
		}
	}
}

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
		prec := 0
		if scaler < 0 {
			prec = int(-scaler)
		}
		scaled := float64(raw) * math.Pow(10, float64(scaler))
		valuePart = fmt.Sprintf("%.*f", prec, scaled)
	}

	unitPart := ""
	if entry.Unit != nil {
		unitPart = sml.UnitName(*entry.Unit)
	}
	return fmt.Sprintf("%s#%s#%s", obis, valuePart, unitPart), true
}

// formatOctetStringMixed uses sml_server's "mixed" algorithm: printable bytes
// (0x21..0x7a) as ASCII, then hex with trailing space once any byte is outside
// that range.
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
