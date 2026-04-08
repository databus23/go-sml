package sml

import "testing"

func TestDLMSUnits(t *testing.T) {
	tests := []struct {
		code uint8
		want string
	}{
		{27, "W"},
		{28, "VA"},
		{29, "var"},
		{30, "Wh"},
		{33, "A"},
		{35, "V"},
		{44, "Hz"},
		{0, ""},   // unknown code
		{58, ""},  // undefined gap
		{100, ""}, // unknown code
	}

	for _, tt := range tests {
		got := UnitName(tt.code)
		if got != tt.want {
			t.Errorf("UnitName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
