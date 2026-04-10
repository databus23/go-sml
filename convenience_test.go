package sml

import (
	"testing"
)

func TestScaledValue(t *testing.T) {
	scaler := func(s int8) *int8 { return &s }

	t.Run("Uint32 with negative scaler", func(t *testing.T) {
		e := &ListEntry{Value: Uint32(12345), Scaler: scaler(-2)}
		got, ok := e.ScaledValue()
		if !ok {
			t.Fatal("ScaledValue returned false, want true")
		}
		if got != 123.45 {
			t.Errorf("ScaledValue() = %v, want 123.45", got)
		}
	})

	t.Run("Int16 with zero scaler", func(t *testing.T) {
		e := &ListEntry{Value: Int16(-100), Scaler: scaler(0)}
		got, ok := e.ScaledValue()
		if !ok {
			t.Fatal("ScaledValue returned false, want true")
		}
		if got != -100.0 {
			t.Errorf("ScaledValue() = %v, want -100.0", got)
		}
	})

	t.Run("non-numeric value returns false", func(t *testing.T) {
		e := &ListEntry{Value: OctetString("hello"), Scaler: scaler(0)}
		_, ok := e.ScaledValue()
		if ok {
			t.Fatal("ScaledValue returned true for non-numeric value, want false")
		}
	})

	t.Run("nil scaler returns false", func(t *testing.T) {
		e := &ListEntry{Value: Uint32(42), Scaler: nil}
		_, ok := e.ScaledValue()
		if ok {
			t.Fatal("ScaledValue returned true for nil scaler, want false")
		}
	})

	t.Run("nil value returns false", func(t *testing.T) {
		e := &ListEntry{Value: nil, Scaler: scaler(0)}
		_, ok := e.ScaledValue()
		if ok {
			t.Fatal("ScaledValue returned true for nil value, want false")
		}
	})
}

func TestOBISString(t *testing.T) {
	t.Run("6-byte OBIS code", func(t *testing.T) {
		e := &ListEntry{ObjName: []byte{1, 0, 1, 8, 0, 255}}
		got := e.OBISString()
		want := "1-0:1.8.0*255"
		if got != want {
			t.Errorf("OBISString() = %q, want %q", got, want)
		}
	})

	t.Run("short input returns empty string", func(t *testing.T) {
		e := &ListEntry{ObjName: []byte{1, 0, 1}}
		got := e.OBISString()
		if got != "" {
			t.Errorf("OBISString() = %q, want \"\"", got)
		}
	})

	t.Run("empty input returns empty string", func(t *testing.T) {
		e := &ListEntry{ObjName: []byte{}}
		got := e.OBISString()
		if got != "" {
			t.Errorf("OBISString() = %q, want \"\"", got)
		}
	})
}

func TestUnitString(t *testing.T) {
	unit := func(u uint8) *uint8 { return &u }

	t.Run("known unit code 30 returns Wh", func(t *testing.T) {
		e := &ListEntry{Unit: unit(30)}
		got := e.UnitString()
		if got != "Wh" {
			t.Errorf("UnitString() = %q, want \"Wh\"", got)
		}
	})

	t.Run("nil unit returns empty string", func(t *testing.T) {
		e := &ListEntry{Unit: nil}
		got := e.UnitString()
		if got != "" {
			t.Errorf("UnitString() = %q, want \"\"", got)
		}
	})

	t.Run("unknown code returns empty string", func(t *testing.T) {
		e := &ListEntry{Unit: unit(100)}
		got := e.UnitString()
		if got != "" {
			t.Errorf("UnitString() = %q, want \"\"", got)
		}
	})
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name string
		val  Value
		want int64
		ok   bool
	}{
		{"Int8", Int8(-42), -42, true},
		{"Int16", Int16(-1000), -1000, true},
		{"Int32", Int32(-100000), -100000, true},
		{"Int64", Int64(-9999999999), -9999999999, true},
		{"Uint8", Uint8(255), 255, true},
		{"Uint16", Uint16(65535), 65535, true},
		{"Uint32", Uint32(4294967295), 4294967295, true},
		{"Uint64", Uint64(1234567890), 1234567890, true},
		{"OctetString returns false", OctetString("hello"), 0, false},
		{"Bool returns false", Bool(true), 0, false},
		{"nil returns false", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt64(tt.val)
			if ok != tt.ok {
				t.Fatalf("ToInt64() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ToInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatReadingValue(t *testing.T) {
	scaler := func(s int8) *int8 { return &s }
	unit := func(u uint8) *uint8 { return &u }

	t.Run("numeric value with scaler and unit", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 1, 8, 0, 255},
			Unit:    unit(30),
			Scaler:  scaler(-1),
			Value:   Uint32(9876),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:1.8.0*255#987.6#Wh"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("numeric value with zero scaler", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 2, 8, 0, 255},
			Unit:    unit(30),
			Scaler:  scaler(0),
			Value:   Uint32(1234),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:2.8.0*255#1234#Wh"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("numeric value with nil scaler", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 1, 8, 0, 255},
			Unit:    unit(30),
			Scaler:  nil,
			Value:   Uint32(42),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:1.8.0*255#42#Wh"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("octet string value", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 96, 1, 0, 255},
			Value:   OctetString([]byte("ABC")),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:96.1.0*255#ABC#"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("octet string with non-printable bytes", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 96, 1, 0, 255},
			Value:   OctetString([]byte{0x41, 0x42, 0x00, 0x43}),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:96.1.0*255#AB00 43 #"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("bool true", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 96, 5, 0, 255},
			Value:   Bool(true),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:96.5.0*255#true#"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("bool false", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 96, 5, 0, 255},
			Value:   Bool(false),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:96.5.0*255#false#"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})

	t.Run("nil value returns false", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 1, 8, 0, 255},
			Value:   nil,
		}
		_, ok := FormatReadingValue(e)
		if ok {
			t.Fatal("FormatReadingValue returned true for nil value, want false")
		}
	})

	t.Run("short OBIS returns false", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 1},
			Value:   Uint32(42),
		}
		_, ok := FormatReadingValue(e)
		if ok {
			t.Fatal("FormatReadingValue returned true for short OBIS, want false")
		}
	})

	t.Run("nil unit produces empty unit part", func(t *testing.T) {
		e := &ListEntry{
			ObjName: []byte{1, 0, 1, 8, 0, 255},
			Unit:    nil,
			Scaler:  scaler(0),
			Value:   Uint32(42),
		}
		got, ok := FormatReadingValue(e)
		if !ok {
			t.Fatal("FormatReadingValue returned false, want true")
		}
		want := "1-0:1.8.0*255#42#"
		if got != want {
			t.Errorf("FormatReadingValue() = %q, want %q", got, want)
		}
	})
}

func TestReadings(t *testing.T) {
	scaler := func(s int8) *int8 { return &s }
	unit := func(u uint8) *uint8 { return &u }

	entries := []ListEntry{
		{
			ObjName: []byte{1, 0, 1, 8, 0, 255},
			Unit:    unit(30),
			Scaler:  scaler(-1),
			Value:   Uint32(9876),
		},
		{
			ObjName: []byte{1, 0, 2, 8, 0, 255},
			Unit:    unit(30),
			Scaler:  scaler(0),
			Value:   Uint32(1234),
		},
	}

	f := &File{
		Messages: []*Message{
			{Body: &OpenResponse{}},
			{Body: &GetListResponse{ValList: entries}},
			{Body: &CloseResponse{}},
		},
	}

	got := f.Readings()
	if len(got) != len(entries) {
		t.Fatalf("Readings() returned %d entries, want %d", len(got), len(entries))
	}
	for i, e := range entries {
		if string(got[i].ObjName) != string(e.ObjName) {
			t.Errorf("Readings()[%d].ObjName = %v, want %v", i, got[i].ObjName, e.ObjName)
		}
	}
}

func TestReadingsMultipleGetList(t *testing.T) {
	entries1 := []ListEntry{{ObjName: []byte{1, 0, 1, 8, 0, 255}}}
	entries2 := []ListEntry{{ObjName: []byte{1, 0, 2, 8, 0, 255}}}

	f := &File{
		Messages: []*Message{
			{Body: &GetListResponse{ValList: entries1}},
			{Body: &GetListResponse{ValList: entries2}},
		},
	}

	got := f.Readings()
	if len(got) != 2 {
		t.Fatalf("Readings() returned %d entries, want 2", len(got))
	}
}

func TestReadingsEmptyFile(t *testing.T) {
	f := &File{}
	got := f.Readings()
	if len(got) != 0 {
		t.Fatalf("Readings() returned %d entries, want 0", len(got))
	}
}
