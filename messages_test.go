package sml

import (
	"bytes"
	"testing"
)

func TestParseOpenResponse(t *testing.T) {
	t.Run("all optionals skipped except sml_version", func(t *testing.T) {
		// 0x76 = list of 6 elements
		// codepage:    0x01 (optional skipped)
		// client_id:   0x01 (optional skipped)
		// req_file_id: 0x07 + 6 bytes (octet string, TL total=7 → data=6)
		// server_id:   0x05 + 4 bytes (octet string, TL total=5 → data=4)
		// ref_time:    0x01 (optional skipped)
		// sml_version: 0x62 0x01 (u8 = 1)
		input := []byte{
			0x76,                               // list of 6
			0x01,                               // codepage: absent
			0x01,                               // client_id: absent
			0x07, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, // req_file_id: 6 bytes
			0x05, 0x0A, 0x0B, 0x0C, 0x0D, // server_id: 4 bytes
			0x01,       // ref_time: absent
			0x62, 0x01, // sml_version: u8(1)
		}
		d := newDecoder(input)
		got := d.readOpenResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil OpenResponse")
		}

		if got.Codepage != nil {
			t.Fatalf("Codepage = %v, want nil", *got.Codepage)
		}
		if got.ClientID != nil {
			t.Fatalf("ClientID = %v, want nil", got.ClientID)
		}

		wantReqFileID := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x01}
		if !bytes.Equal(got.ReqFileID, wantReqFileID) {
			t.Fatalf("ReqFileID = %X, want %X", got.ReqFileID, wantReqFileID)
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		if got.RefTime != nil {
			t.Fatalf("RefTime = %v, want nil", got.RefTime)
		}

		if got.SmlVersion == nil {
			t.Fatal("SmlVersion = nil, want non-nil")
		}
		if *got.SmlVersion != 1 {
			t.Fatalf("SmlVersion = %d, want 1", *got.SmlVersion)
		}
	})

	t.Run("codepage and ref_time present", func(t *testing.T) {
		// 0x76 = list of 6
		// codepage:    0x04 0x55 0x54 0x46 (octet string "UTF")
		// client_id:   0x01 (absent)
		// req_file_id: 0x03 0xAA 0xBB (2 bytes)
		// server_id:   0x03 0xCC 0xDD (2 bytes)
		// ref_time:    0x72 0x62 0x01 0x65 0x00 0x00 0x30 0x39 (SecIndex=12345)
		// sml_version: 0x01 (absent)
		input := []byte{
			0x76,                   // list of 6
			0x04, 0x55, 0x54, 0x46, // codepage: "UTF"
			0x01,             // client_id: absent
			0x03, 0xAA, 0xBB, // req_file_id: 2 bytes
			0x03, 0xCC, 0xDD, // server_id: 2 bytes
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39, // ref_time: SecIndex=12345
			0x01, // sml_version: absent
		}
		d := newDecoder(input)
		got := d.readOpenResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}

		if got.Codepage == nil {
			t.Fatal("Codepage = nil, want non-nil")
		}
		if *got.Codepage != "UTF" {
			t.Fatalf("Codepage = %q, want %q", *got.Codepage, "UTF")
		}

		if got.ClientID != nil {
			t.Fatalf("ClientID = %v, want nil", got.ClientID)
		}

		if got.RefTime == nil {
			t.Fatal("RefTime = nil, want non-nil")
		}
		if got.RefTime.Tag != TimeSecIndex {
			t.Fatalf("RefTime.Tag = %d, want %d (TimeSecIndex)", got.RefTime.Tag, TimeSecIndex)
		}
		if got.RefTime.Value != 12345 {
			t.Fatalf("RefTime.Value = %d, want 12345", got.RefTime.Value)
		}

		if got.SmlVersion != nil {
			t.Fatalf("SmlVersion = %v, want nil", *got.SmlVersion)
		}
	})
}

func TestParseCloseResponse(t *testing.T) {
	t.Run("signature absent", func(t *testing.T) {
		// 0x71 = list of 1 element
		// signature: 0x01 (optional skipped)
		input := []byte{0x71, 0x01}
		d := newDecoder(input)
		got := d.readCloseResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil CloseResponse")
		}
		if got.Signature != nil {
			t.Fatalf("Signature = %v, want nil", got.Signature)
		}
	})

	t.Run("signature present", func(t *testing.T) {
		// 0x71 = list of 1 element
		// signature: 0x03 0xAA 0xBB (octet string 2 bytes)
		input := []byte{0x71, 0x03, 0xAA, 0xBB}
		d := newDecoder(input)
		got := d.readCloseResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil CloseResponse")
		}
		want := []byte{0xAA, 0xBB}
		if !bytes.Equal(got.Signature, want) {
			t.Fatalf("Signature = %X, want %X", got.Signature, want)
		}
	})
}
