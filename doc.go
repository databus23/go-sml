// Package sml decodes Smart Message Language (SML) data from German smart
// electricity meters.
//
// SML is a binary TLV-encoded protocol defined by the German BSI. It is used
// by virtually all modern German smart meters to transmit readings over an
// infrared optical interface. This package provides a pure-Go parser with zero
// external dependencies.
//
// # Decoding a single payload
//
// If you already have a complete SML payload (transport layer stripped), use
// [Decode] or [DecodeWithOptions]:
//
//	file, err := sml.Decode(payload)
//	for _, entry := range file.Readings() {
//	    if v, ok := entry.ScaledValue(); ok {
//	        fmt.Printf("%s = %f %s\n", entry.OBISString(), v, entry.UnitString())
//	    }
//	}
//
// # Streaming from an io.Reader
//
// For continuous reading from a serial port or other byte stream, use [Listen]
// or [ListenMessages]. These handle SML transport framing (start/end
// sequences, escape processing, CRC validation) automatically via the
// [transport] sub-package:
//
//	err := sml.Listen(ctx, serialPort, func(file *sml.File) error {
//	    for _, entry := range file.Readings() {
//	        // process readings
//	    }
//	    return nil
//	})
//
// # Value types
//
// SML values preserve their wire encoding width. The [Value] interface is
// satisfied by named types ([OctetString], [Bool], [Int8], [Int16], [Int32],
// [Int64], [Uint8], [Uint16], [Uint32], [Uint64]) which can be distinguished
// via a type switch. For meter readings, the [ListEntry.ScaledValue]
// convenience method applies the scaler and returns a float64.
package sml
