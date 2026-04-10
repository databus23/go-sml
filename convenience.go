package sml

import (
	"fmt"
	"math"
	"strings"
)

// ToInt64 extracts the numeric value from a Value as int64.
// Returns (0, false) for nil, OctetString, and Bool values.
func ToInt64(v Value) (int64, bool) {
	switch val := v.(type) {
	case Int8:
		return int64(val), true
	case Int16:
		return int64(val), true
	case Int32:
		return int64(val), true
	case Int64:
		return int64(val), true
	case Uint8:
		return int64(val), true
	case Uint16:
		return int64(val), true
	case Uint32:
		return int64(val), true
	case Uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

// ScaledValue returns the numeric value multiplied by 10^Scaler as a float64.
// Returns (0, false) if Value is nil, non-numeric, or Scaler is nil.
func (e *ListEntry) ScaledValue() (float64, bool) {
	if e.Scaler == nil || e.Value == nil {
		return 0, false
	}
	raw, ok := ToInt64(e.Value)
	if !ok {
		return 0, false
	}
	return float64(raw) * math.Pow(10, float64(*e.Scaler)), true
}

// OBISString formats the 6-byte OBIS code as "A-B:C.D.E*F".
// Returns "" if ObjName is not exactly 6 bytes.
func (e *ListEntry) OBISString() string {
	if len(e.ObjName) != 6 {
		return ""
	}
	return fmt.Sprintf("%d-%d:%d.%d.%d*%d",
		e.ObjName[0], e.ObjName[1], e.ObjName[2],
		e.ObjName[3], e.ObjName[4], e.ObjName[5])
}

// UnitString returns the DLMS unit name for the entry's unit code.
// Returns "" if Unit is nil or the code is unknown.
func (e *ListEntry) UnitString() string {
	if e.Unit == nil {
		return ""
	}
	return UnitName(*e.Unit)
}

// Readings collects all ListEntry values from every GetListResponse in the file,
// returning them as a flat slice.
func (f *File) Readings() []ListEntry {
	var result []ListEntry
	for _, msg := range f.Messages {
		if glr, ok := msg.Body.(*GetListResponse); ok {
			result = append(result, glr.ValList...)
		}
	}
	return result
}

// FormatReadingValue formats a ListEntry as "OBIS#value#unit", matching the
// output format of the C sml_server reference tool. Returns ("", false) if the
// entry has a nil Value or an OBIS code that isn't exactly 6 bytes.
func FormatReadingValue(e *ListEntry) (string, bool) {
	if e.Value == nil {
		return "", false
	}
	obis := e.OBISString()
	if obis == "" {
		return "", false
	}

	var valuePart string
	switch val := e.Value.(type) {
	case OctetString:
		valuePart = formatOctetStringMixed([]byte(val))
	case Bool:
		if bool(val) {
			valuePart = "true"
		} else {
			valuePart = "false"
		}
	default:
		raw, ok := ToInt64(e.Value)
		if !ok {
			return "", false
		}
		var scaler int8
		if e.Scaler != nil {
			scaler = *e.Scaler
		}
		precision := 0
		if scaler < 0 {
			precision = int(-scaler)
		}
		scaled := float64(raw) * math.Pow(10, float64(scaler))
		valuePart = fmt.Sprintf("%.*f", precision, scaled)
	}

	unitPart := ""
	if e.Unit != nil {
		unitPart = UnitName(*e.Unit)
	}
	return fmt.Sprintf("%s#%s#%s", obis, valuePart, unitPart), true
}

// formatOctetStringMixed formats an OctetString using the C sml_server "mixed"
// algorithm: bytes in range (0x20, 0x7b) are output as ASCII; once any byte
// falls outside that range, all remaining bytes use lowercase hex with trailing space.
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
