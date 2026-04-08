// Package transport handles SML transport layer framing.
//
// SML data on the wire is wrapped in a transport frame consisting of a start
// sequence (1b1b1b1b 01010101), the escaped payload, an end sequence
// (1b1b1b1b 1aXXYYZZ), and a CRC-16 checksum. [Reader] extracts and validates
// these frames from any [io.Reader], such as a serial port.
//
// CRC auto-detection transparently supports both the standard CRC-16/X.25
// variant and the CRC-16/Kermit variant used by Holley DTZ541 meters.
//
// Most users do not need to use this package directly — the top-level
// [sml.Listen] and [sml.ListenMessages] functions compose transport reading
// with SML decoding automatically.
package transport
