package transport

import "testing"

func TestCRC16Table(t *testing.T) {
	// First 16 entries of the reflected CRC-CCITT lookup table (polynomial 0x8408).
	want := [16]uint16{
		0x0000, 0x1189, 0x2312, 0x329b,
		0x4624, 0x57ad, 0x6536, 0x74bf,
		0x8c48, 0x9dc1, 0xaf5a, 0xbed3,
		0xca6c, 0xdbe5, 0xe97e, 0xf8f7,
	}
	for i, w := range want {
		if crcTable[i] != w {
			t.Errorf("crcTable[%d] = 0x%04x, want 0x%04x", i, crcTable[i], w)
		}
	}
}

func TestCRC16(t *testing.T) {
	t.Run("SML start sequence", func(t *testing.T) {
		// 1b1b1b1b 01010101
		data := []byte{0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01}
		got := crc16(data)
		want := uint16(0x236e)
		if got != want {
			t.Fatalf("crc16(SML start) = 0x%04x, want 0x%04x", got, want)
		}
	})

	t.Run("check value for 123456789", func(t *testing.T) {
		// Standard CRC-16/X.25 check value is 0x906E, but our crc16 includes
		// the DIN 62056-46 byte-swap, producing 0x6E90.
		data := []byte("123456789")
		got := crc16(data)
		want := uint16(0x6e90)
		if got != want {
			t.Fatalf("crc16(\"123456789\") = 0x%04x, want 0x%04x", got, want)
		}
	})
}

func TestCRC16Kermit(t *testing.T) {
	t.Run("SML start sequence", func(t *testing.T) {
		data := []byte{0x1b, 0x1b, 0x1b, 0x1b, 0x01, 0x01, 0x01, 0x01}
		got := crc16Kermit(data)
		want := uint16(0xed50)
		if got != want {
			t.Fatalf("crc16Kermit(SML start) = 0x%04x, want 0x%04x", got, want)
		}
	})

	t.Run("check value for 123456789", func(t *testing.T) {
		// Standard CRC-16/Kermit check value for "123456789" is 0x2189.
		data := []byte("123456789")
		got := crc16Kermit(data)
		want := uint16(0x2189)
		if got != want {
			t.Fatalf("crc16Kermit(\"123456789\") = 0x%04x, want 0x%04x", got, want)
		}
	})
}

func TestCRC16EmptyInput(t *testing.T) {
	t.Run("X.25 empty", func(t *testing.T) {
		// init=0xFFFF, no iterations, XOR→0x0000, byte-swap→0x0000
		got := crc16(nil)
		want := uint16(0x0000)
		if got != want {
			t.Fatalf("crc16(nil) = 0x%04x, want 0x%04x", got, want)
		}
	})

	t.Run("Kermit empty", func(t *testing.T) {
		// init=0x0000, no iterations, no XOR, no swap → 0x0000
		got := crc16Kermit(nil)
		want := uint16(0x0000)
		if got != want {
			t.Fatalf("crc16Kermit(nil) = 0x%04x, want 0x%04x", got, want)
		}
	})

	t.Run("X.25 empty slice", func(t *testing.T) {
		got := crc16([]byte{})
		want := uint16(0x0000)
		if got != want {
			t.Fatalf("crc16([]byte{}) = 0x%04x, want 0x%04x", got, want)
		}
	})

	t.Run("Kermit empty slice", func(t *testing.T) {
		got := crc16Kermit([]byte{})
		want := uint16(0x0000)
		if got != want {
			t.Fatalf("crc16Kermit([]byte{}) = 0x%04x, want 0x%04x", got, want)
		}
	})
}
