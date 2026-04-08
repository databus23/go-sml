package sml

import (
	"fmt"
	"math"
)

// ScaledValue returns the numeric value multiplied by 10^Scaler as a float64.
// Returns (0, false) if Value is nil, non-numeric, or Scaler is nil.
func (e *ListEntry) ScaledValue() (float64, bool) {
	if e.Scaler == nil || e.Value == nil {
		return 0, false
	}
	var v float64
	switch val := e.Value.(type) {
	case Int8:
		v = float64(val)
	case Int16:
		v = float64(val)
	case Int32:
		v = float64(val)
	case Int64:
		v = float64(val)
	case Uint8:
		v = float64(val)
	case Uint16:
		v = float64(val)
	case Uint32:
		v = float64(val)
	case Uint64:
		v = float64(val)
	default:
		return 0, false
	}
	return v * math.Pow(10, float64(*e.Scaler)), true
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
