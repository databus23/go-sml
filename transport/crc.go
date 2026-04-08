package transport

// crcTable is the reflected CRC-CCITT lookup table (polynomial 0x8408).
// Each entry is computed by iterating 8 bits with the reflected polynomial.
var crcTable = func() [256]uint16 {
	var t [256]uint16
	for i := range t {
		crc := uint16(i)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0x8408
			} else {
				crc >>= 1
			}
		}
		t[i] = crc
	}
	return t
}()

// crc16 computes the CRC-16/X.25 checksum with the DIN 62056-46 byte-swap.
// Init: 0xFFFF, polynomial: 0x8408 (reflected), final: XOR 0xFFFF + byte-swap.
func crc16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc = (crc >> 8) ^ crcTable[(crc^uint16(b))&0xFF]
	}
	crc ^= 0xFFFF
	// Byte-swap: little-endian to big-endian
	return (crc << 8) | (crc >> 8)
}

// crc16Kermit computes the CRC-16/Kermit checksum (Holley workaround variant).
// Init: 0x0000, polynomial: 0x8408 (reflected), no final XOR, no byte-swap.
func crc16Kermit(data []byte) uint16 {
	crc := uint16(0x0000)
	for _, b := range data {
		crc = (crc >> 8) ^ crcTable[(crc^uint16(b))&0xFF]
	}
	return crc
}
