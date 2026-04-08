package sml

import (
	"bytes"
	"testing"
)

// buildOpenResponseMsg constructs a minimal SML message wrapping an OpenResponse.
// The message is a 6-element list (0x76):
//  1. transaction_id: octet string
//  2. group_id: u8
//  3. abort_on_error: u8
//  4. message_body: 2-element list (tag u32 + OpenResponse data)
//  5. crc: u16
//  6. end_of_msg: 0x00
func buildOpenResponseMsg(txID byte) []byte {
	return []byte{
		0x76,             // list of 6
		0x02, txID,       // transaction_id: 1-byte octet string
		0x62, 0x00,       // group_id: u8(0)
		0x62, 0x00,       // abort_on_error: u8(0)
		0x72,             // message_body: list of 2
		0x65, 0x00, 0x00, 0x01, 0x01, // tag: u32(0x00000101) = OpenResponse
		// OpenResponse data: 6-element list
		0x76,                               // list of 6
		0x01,                               // codepage: absent
		0x01,                               // client_id: absent
		0x03, 0xAA, 0xBB,                   // req_file_id: 2 bytes
		0x03, 0xCC, 0xDD,                   // server_id: 2 bytes
		0x01,                               // ref_time: absent
		0x01,                               // sml_version: absent
		0x63, 0x00, 0x00, // crc: u16(0)
		0x00, // end_of_msg
	}
}

// buildCloseResponseMsg constructs a minimal SML message wrapping a CloseResponse.
func buildCloseResponseMsg(txID byte) []byte {
	return []byte{
		0x76,             // list of 6
		0x02, txID,       // transaction_id: 1-byte octet string
		0x62, 0x00,       // group_id: u8(0)
		0x62, 0x00,       // abort_on_error: u8(0)
		0x72,             // message_body: list of 2
		0x65, 0x00, 0x00, 0x02, 0x01, // tag: u32(0x00000201) = CloseResponse
		// CloseResponse data: 1-element list
		0x71,       // list of 1
		0x01,       // signature: absent
		0x63, 0x00, 0x00, // crc: u16(0)
		0x00, // end_of_msg
	}
}

// buildGetListResponseMsg constructs a minimal SML message wrapping a GetListResponse
// with zero value list entries.
func buildGetListResponseMsg(txID byte) []byte {
	return []byte{
		0x76,             // list of 6
		0x02, txID,       // transaction_id: 1-byte octet string
		0x62, 0x00,       // group_id: u8(0)
		0x62, 0x00,       // abort_on_error: u8(0)
		0x72,             // message_body: list of 2
		0x65, 0x00, 0x00, 0x07, 0x01, // tag: u32(0x00000701) = GetListResponse
		// GetListResponse data: 7-element list
		0x77,                          // list of 7
		0x01,                          // client_id: absent
		0x03, 0x0A, 0x0B,             // server_id: 2 bytes
		0x01,                          // list_name: absent
		0x01,                          // act_sensor_time: absent
		0x70,                          // val_list: list of 0
		0x01,                          // signature: absent
		0x01,                          // act_gateway_time: absent
		0x63, 0x00, 0x00, // crc: u16(0)
		0x00, // end_of_msg
	}
}

// buildRequestMsg constructs a minimal SML message with a request tag (e.g., OpenRequest = 0x00000100).
// The body data is an empty list (0x70) — just enough to be parseable but tagged as a request.
func buildRequestMsg(txID byte) []byte {
	return []byte{
		0x76,             // list of 6
		0x02, txID,       // transaction_id: 1-byte octet string
		0x62, 0x00,       // group_id: u8(0)
		0x62, 0x00,       // abort_on_error: u8(0)
		0x72,             // message_body: list of 2
		0x65, 0x00, 0x00, 0x01, 0x00, // tag: u32(0x00000100) = OpenRequest (not a response)
		0x70,             // body: empty list (we skip it anyway)
		0x63, 0x00, 0x00, // crc: u16(0)
		0x00, // end_of_msg
	}
}

func TestDecode(t *testing.T) {
	// Build a file with 3 messages: Open, GetList, Close
	var buf []byte
	buf = append(buf, buildOpenResponseMsg(0x01)...)
	buf = append(buf, buildGetListResponseMsg(0x02)...)
	buf = append(buf, buildCloseResponseMsg(0x03)...)

	file, err := Decode(buf)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if file == nil {
		t.Fatal("Decode returned nil File")
	}
	if len(file.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(file.Messages))
	}

	// Message 0: OpenResponse
	if _, ok := file.Messages[0].Body.(*OpenResponse); !ok {
		t.Fatalf("Messages[0].Body is %T, want *OpenResponse", file.Messages[0].Body)
	}
	if !bytes.Equal(file.Messages[0].TransactionID, []byte{0x01}) {
		t.Fatalf("Messages[0].TransactionID = %X, want 01", file.Messages[0].TransactionID)
	}

	// Message 1: GetListResponse
	glr, ok := file.Messages[1].Body.(*GetListResponse)
	if !ok {
		t.Fatalf("Messages[1].Body is %T, want *GetListResponse", file.Messages[1].Body)
	}
	wantServerID := []byte{0x0A, 0x0B}
	if !bytes.Equal(glr.ServerID, wantServerID) {
		t.Fatalf("GetListResponse.ServerID = %X, want %X", glr.ServerID, wantServerID)
	}

	// Message 2: CloseResponse
	if _, ok := file.Messages[2].Body.(*CloseResponse); !ok {
		t.Fatalf("Messages[2].Body is %T, want *CloseResponse", file.Messages[2].Body)
	}
}

func TestDecodeSkipsRequestMessages(t *testing.T) {
	// Build: Open (response) + OpenRequest (request, should be skipped) + Close (response)
	var buf []byte
	buf = append(buf, buildOpenResponseMsg(0x01)...)
	buf = append(buf, buildRequestMsg(0x02)...)
	buf = append(buf, buildCloseResponseMsg(0x03)...)

	file, err := Decode(buf)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if file == nil {
		t.Fatal("Decode returned nil File")
	}
	// The request message should be skipped, leaving only Open + Close
	if len(file.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(file.Messages))
	}
	if _, ok := file.Messages[0].Body.(*OpenResponse); !ok {
		t.Fatalf("Messages[0].Body is %T, want *OpenResponse", file.Messages[0].Body)
	}
	if _, ok := file.Messages[1].Body.(*CloseResponse); !ok {
		t.Fatalf("Messages[1].Body is %T, want *CloseResponse", file.Messages[1].Body)
	}
}

func TestDecodeStrictError(t *testing.T) {
	// Malformed data: starts with a list-6 header but truncated
	malformed := []byte{0x76, 0x02, 0x01, 0x62} // truncated mid-message

	file, err := Decode(malformed)
	if err == nil {
		t.Fatal("Decode should return error for malformed data")
	}
	if file != nil {
		t.Fatalf("Decode should return nil File on strict error, got %+v", file)
	}
}

func TestDecodeNonStrict(t *testing.T) {
	// First message: malformed (truncated OpenResponse)
	malformed := []byte{
		0x76,             // list of 6
		0x02, 0x01,       // transaction_id
		0x62, 0x00,       // group_id
		0x62, 0x00,       // abort_on_error
		0x72,             // message_body: list of 2
		0x65, 0x00, 0x00, 0x01, 0x01, // tag: OpenResponse
		// OpenResponse data: truncated — only partial list header
		0x76, // list of 6 but no fields follow
		// No CRC, no end_of_msg — malformed
	}

	// Second message: valid CloseResponse
	valid := buildCloseResponseMsg(0x02)

	var buf []byte
	buf = append(buf, malformed...)
	buf = append(buf, valid...)

	file, err := DecodeWithOptions(buf, DecodeOptions{Strict: false})
	// Should have a partial result and an error
	if err == nil {
		t.Fatal("DecodeWithOptions non-strict should return error for malformed message")
	}
	if file == nil {
		t.Fatal("DecodeWithOptions non-strict should return partial File, got nil")
	}
	// The valid CloseResponse should have been parsed
	if len(file.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1 (the valid CloseResponse)", len(file.Messages))
	}
	if _, ok := file.Messages[0].Body.(*CloseResponse); !ok {
		t.Fatalf("Messages[0].Body is %T, want *CloseResponse", file.Messages[0].Body)
	}
}

func TestDecodeWithTrailingZeros(t *testing.T) {
	// An SML file with trailing 0x00 bytes between messages and at the end
	var buf []byte
	buf = append(buf, buildOpenResponseMsg(0x01)...)
	buf = append(buf, 0x00, 0x00) // extra trailing zeros
	buf = append(buf, buildCloseResponseMsg(0x02)...)
	buf = append(buf, 0x00, 0x00, 0x00) // trailing zeros at end

	file, err := Decode(buf)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(file.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(file.Messages))
	}
}

func TestDecodeEmptyInput(t *testing.T) {
	file, err := Decode([]byte{})
	if err != nil {
		t.Fatalf("Decode returned error for empty input: %v", err)
	}
	if file == nil {
		t.Fatal("Decode returned nil File for empty input")
	}
	if len(file.Messages) != 0 {
		t.Fatalf("len(Messages) = %d, want 0", len(file.Messages))
	}
}

func TestDecodeOnlyZeros(t *testing.T) {
	// Input is only end-of-message markers
	file, err := Decode([]byte{0x00, 0x00, 0x00})
	if err != nil {
		t.Fatalf("Decode returned error for only-zeros input: %v", err)
	}
	if len(file.Messages) != 0 {
		t.Fatalf("len(Messages) = %d, want 0", len(file.Messages))
	}
}
