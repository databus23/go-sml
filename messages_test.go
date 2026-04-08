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

func TestParseListEntry(t *testing.T) {
	t.Run("all fields present", func(t *testing.T) {
		// 7-element list: 0x77
		// obj_name:  0x07 0x01 0x00 0x01 0x08 0x00 0xFF  (TL=0x07 → 6 bytes, OBIS 1-0:1.8.0*255)
		// status:    0x65 0x00 0x01 0xE2 0x40            (unsigned u32, 4 bytes, value=123456)
		// val_time:  0x72 0x62 0x01 0x65 0x00 0x00 0x30 0x39 (2-element list, SecIndex=12345)
		// unit:      0x62 0x1E                            (unsigned u8, value=30=Wh)
		// scaler:    0x52 0xFE                            (signed i8, value=-2)
		// value:     0x65 0x00 0x01 0xE2 0x40             (unsigned u32, value=123456)
		// signature: 0x01                                 (absent)
		input := []byte{
			0x77,                                     // list of 7
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF, // obj_name: 6 bytes
			0x65, 0x00, 0x01, 0xE2, 0x40,            // status: u32=123456
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39, // val_time: SecIndex=12345
			0x62, 0x1E,                              // unit: u8=30
			0x52, 0xFE,                              // scaler: i8=-2
			0x65, 0x00, 0x01, 0xE2, 0x40,            // value: u32=123456
			0x01,                                    // signature: absent
		}
		d := newDecoder(input)
		got := d.readListEntry()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}

		wantObjName := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(got.ObjName, wantObjName) {
			t.Fatalf("ObjName = %X, want %X", got.ObjName, wantObjName)
		}

		if got.Status == nil {
			t.Fatal("Status = nil, want non-nil")
		}
		if *got.Status != 123456 {
			t.Fatalf("Status = %d, want 123456", *got.Status)
		}

		if got.ValTime == nil {
			t.Fatal("ValTime = nil, want non-nil")
		}
		if got.ValTime.Tag != TimeSecIndex {
			t.Fatalf("ValTime.Tag = %d, want %d (TimeSecIndex)", got.ValTime.Tag, TimeSecIndex)
		}
		if got.ValTime.Value != 12345 {
			t.Fatalf("ValTime.Value = %d, want 12345", got.ValTime.Value)
		}

		if got.Unit == nil {
			t.Fatal("Unit = nil, want non-nil")
		}
		if *got.Unit != 30 {
			t.Fatalf("Unit = %d, want 30", *got.Unit)
		}

		if got.Scaler == nil {
			t.Fatal("Scaler = nil, want non-nil")
		}
		if *got.Scaler != -2 {
			t.Fatalf("Scaler = %d, want -2", *got.Scaler)
		}

		if got.Value == nil {
			t.Fatal("Value = nil, want non-nil")
		}
		if got.Value != Uint32(123456) {
			t.Fatalf("Value = %v, want Uint32(123456)", got.Value)
		}

		if got.Signature != nil {
			t.Fatalf("Signature = %v, want nil", got.Signature)
		}
	})

	t.Run("all optional fields absent", func(t *testing.T) {
		// 7-element list: 0x77
		// obj_name:  0x05 0xAA 0xBB 0xCC 0xDD  (TL=0x05 → 4 bytes)
		// status:    0x01 (absent)
		// val_time:  0x01 (absent)
		// unit:      0x01 (absent)
		// scaler:    0x01 (absent)
		// value:     0x01 (absent)
		// signature: 0x01 (absent)
		input := []byte{
			0x77,                         // list of 7
			0x05, 0xAA, 0xBB, 0xCC, 0xDD, // obj_name: 4 bytes
			0x01,                         // status: absent
			0x01,                         // val_time: absent
			0x01,                         // unit: absent
			0x01,                         // scaler: absent
			0x01,                         // value: absent
			0x01,                         // signature: absent
		}
		d := newDecoder(input)
		got := d.readListEntry()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}

		wantObjName := []byte{0xAA, 0xBB, 0xCC, 0xDD}
		if !bytes.Equal(got.ObjName, wantObjName) {
			t.Fatalf("ObjName = %X, want %X", got.ObjName, wantObjName)
		}
		if got.Status != nil {
			t.Fatalf("Status = %v, want nil", *got.Status)
		}
		if got.ValTime != nil {
			t.Fatalf("ValTime = %v, want nil", got.ValTime)
		}
		if got.Unit != nil {
			t.Fatalf("Unit = %v, want nil", *got.Unit)
		}
		if got.Scaler != nil {
			t.Fatalf("Scaler = %v, want nil", *got.Scaler)
		}
		if got.Value != nil {
			t.Fatalf("Value = %v, want nil", got.Value)
		}
		if got.Signature != nil {
			t.Fatalf("Signature = %v, want nil", got.Signature)
		}
	})
}

func TestParseGetListResponse(t *testing.T) {
	t.Run("two entries, optionals mixed", func(t *testing.T) {
		// GetListResponse = 7-element list (0x77)
		// client_id:       0x01 (absent)
		// server_id:       0x05 0x0A 0x0B 0x0C 0x0D  (4 bytes)
		// list_name:       0x04 0x4C 0x4E 0x41       (3 bytes "LNA")
		// act_sensor_time: 0x72 0x62 0x01 0x65 0x00 0x00 0x07 0xD0 (SecIndex=2000)
		// val_list:        0x72 followed by two 0x77 entries
		//   entry 1 (all fields):
		//     obj_name:  0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		//     status:    0x65 0x00 0x01 0xE2 0x40  (u32=123456)
		//     val_time:  0x01 (absent)
		//     unit:      0x62 0x1E (u8=30)
		//     scaler:    0x52 0xFE (i8=-2)
		//     value:     0x65 0x00 0x01 0xE2 0x40  (u32=123456)
		//     signature: 0x01 (absent)
		//   entry 2 (minimal):
		//     obj_name:  0x05 0xAA 0xBB 0xCC 0xDD
		//     status:    0x01 (absent)
		//     val_time:  0x01 (absent)
		//     unit:      0x01 (absent)
		//     scaler:    0x01 (absent)
		//     value:     0x01 (absent)
		//     signature: 0x01 (absent)
		// list_signature:  0x01 (absent)
		// act_gateway_time:0x01 (absent)
		input := []byte{
			0x77,                               // GetListResponse list of 7
			0x01,                               // client_id: absent
			0x05, 0x0A, 0x0B, 0x0C, 0x0D,      // server_id: 4 bytes
			0x04, 0x4C, 0x4E, 0x41,            // list_name: "LNA"
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x07, 0xD0, // act_sensor_time: SecIndex=2000
			// val_list: 2-element list
			0x72,
			// entry 1 (7 fields)
			0x77,
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF, // obj_name
			0x65, 0x00, 0x01, 0xE2, 0x40,              // status: u32=123456
			0x01,                                       // val_time: absent
			0x62, 0x1E,                                 // unit: u8=30
			0x52, 0xFE,                                 // scaler: i8=-2
			0x65, 0x00, 0x01, 0xE2, 0x40,              // value: u32=123456
			0x01,                                       // signature: absent
			// entry 2 (minimal)
			0x77,
			0x05, 0xAA, 0xBB, 0xCC, 0xDD,              // obj_name: 4 bytes
			0x01,                                       // status: absent
			0x01,                                       // val_time: absent
			0x01,                                       // unit: absent
			0x01,                                       // scaler: absent
			0x01,                                       // value: absent
			0x01,                                       // signature: absent
			// list_signature: absent
			0x01,
			// act_gateway_time: absent
			0x01,
		}
		d := newDecoder(input)
		got := d.readGetListResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetListResponse")
		}

		if got.ClientID != nil {
			t.Fatalf("ClientID = %v, want nil", got.ClientID)
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		wantListName := []byte{0x4C, 0x4E, 0x41}
		if !bytes.Equal(got.ListName, wantListName) {
			t.Fatalf("ListName = %X, want %X", got.ListName, wantListName)
		}

		if got.ActSensorTime == nil {
			t.Fatal("ActSensorTime = nil, want non-nil")
		}
		if got.ActSensorTime.Tag != TimeSecIndex {
			t.Fatalf("ActSensorTime.Tag = %d, want %d", got.ActSensorTime.Tag, TimeSecIndex)
		}
		if got.ActSensorTime.Value != 2000 {
			t.Fatalf("ActSensorTime.Value = %d, want 2000", got.ActSensorTime.Value)
		}

		if len(got.ValList) != 2 {
			t.Fatalf("len(ValList) = %d, want 2", len(got.ValList))
		}

		e1 := got.ValList[0]
		wantOBIS := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(e1.ObjName, wantOBIS) {
			t.Fatalf("ValList[0].ObjName = %X, want %X", e1.ObjName, wantOBIS)
		}
		if e1.Status == nil {
			t.Fatal("ValList[0].Status = nil, want non-nil")
		}
		if *e1.Status != 123456 {
			t.Fatalf("ValList[0].Status = %d, want 123456", *e1.Status)
		}
		if e1.ValTime != nil {
			t.Fatalf("ValList[0].ValTime = %v, want nil", e1.ValTime)
		}
		if e1.Unit == nil {
			t.Fatal("ValList[0].Unit = nil, want non-nil")
		}
		if *e1.Unit != 30 {
			t.Fatalf("ValList[0].Unit = %d, want 30", *e1.Unit)
		}
		if e1.Scaler == nil {
			t.Fatal("ValList[0].Scaler = nil, want non-nil")
		}
		if *e1.Scaler != -2 {
			t.Fatalf("ValList[0].Scaler = %d, want -2", *e1.Scaler)
		}
		if e1.Value != Uint32(123456) {
			t.Fatalf("ValList[0].Value = %v, want Uint32(123456)", e1.Value)
		}
		if e1.Signature != nil {
			t.Fatalf("ValList[0].Signature = %v, want nil", e1.Signature)
		}

		e2 := got.ValList[1]
		wantObjName2 := []byte{0xAA, 0xBB, 0xCC, 0xDD}
		if !bytes.Equal(e2.ObjName, wantObjName2) {
			t.Fatalf("ValList[1].ObjName = %X, want %X", e2.ObjName, wantObjName2)
		}
		if e2.Status != nil {
			t.Fatalf("ValList[1].Status = %v, want nil", *e2.Status)
		}
		if e2.Value != nil {
			t.Fatalf("ValList[1].Value = %v, want nil", e2.Value)
		}

		if got.Signature != nil {
			t.Fatalf("Signature = %v, want nil", got.Signature)
		}
		if got.ActGatewayTime != nil {
			t.Fatalf("ActGatewayTime = %v, want nil", got.ActGatewayTime)
		}
	})

	t.Run("client_id and act_gateway_time present", func(t *testing.T) {
		// GetListResponse = 7-element list (0x77)
		// client_id:       0x03 0x11 0x22  (2 bytes)
		// server_id:       0x03 0x33 0x44  (2 bytes)
		// list_name:       0x01 (absent)
		// act_sensor_time: 0x01 (absent)
		// val_list:        0x71 followed by one 0x77 entry
		//   entry (minimal):
		//     obj_name:  0x03 0x55 0x66   (2 bytes)
		//     status..signature: all 0x01
		// list_signature:  0x03 0xDE 0xAD  (2 bytes)
		// act_gateway_time:0x72 0x62 0x01 0x65 0x00 0x00 0x13 0x88 (SecIndex=5000)
		input := []byte{
			0x77,                               // GetListResponse list of 7
			0x03, 0x11, 0x22,                   // client_id: 2 bytes
			0x03, 0x33, 0x44,                   // server_id: 2 bytes
			0x01,                               // list_name: absent
			0x01,                               // act_sensor_time: absent
			// val_list: 1-element list
			0x71,
			// entry (minimal)
			0x77,
			0x03, 0x55, 0x66,                   // obj_name: 2 bytes
			0x01,                               // status: absent
			0x01,                               // val_time: absent
			0x01,                               // unit: absent
			0x01,                               // scaler: absent
			0x01,                               // value: absent
			0x01,                               // signature: absent
			// list_signature: 2 bytes
			0x03, 0xDE, 0xAD,
			// act_gateway_time: SecIndex=5000
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x13, 0x88,
		}
		d := newDecoder(input)
		got := d.readGetListResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetListResponse")
		}

		wantClientID := []byte{0x11, 0x22}
		if !bytes.Equal(got.ClientID, wantClientID) {
			t.Fatalf("ClientID = %X, want %X", got.ClientID, wantClientID)
		}

		wantServerID := []byte{0x33, 0x44}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		if got.ListName != nil {
			t.Fatalf("ListName = %v, want nil", got.ListName)
		}
		if got.ActSensorTime != nil {
			t.Fatalf("ActSensorTime = %v, want nil", got.ActSensorTime)
		}

		if len(got.ValList) != 1 {
			t.Fatalf("len(ValList) = %d, want 1", len(got.ValList))
		}

		wantSig := []byte{0xDE, 0xAD}
		if !bytes.Equal(got.Signature, wantSig) {
			t.Fatalf("Signature = %X, want %X", got.Signature, wantSig)
		}

		if got.ActGatewayTime == nil {
			t.Fatal("ActGatewayTime = nil, want non-nil")
		}
		if got.ActGatewayTime.Tag != TimeSecIndex {
			t.Fatalf("ActGatewayTime.Tag = %d, want %d", got.ActGatewayTime.Tag, TimeSecIndex)
		}
		if got.ActGatewayTime.Value != 5000 {
			t.Fatalf("ActGatewayTime.Value = %d, want 5000", got.ActGatewayTime.Value)
		}
	})
}

func TestParseAttentionResponse(t *testing.T) {
	t.Run("all fields present with details", func(t *testing.T) {
		// AttentionResponse = 4-element list (0x74)
		// server_id:         0x05 0x0A 0x0B 0x0C 0x0D  (4 bytes)
		// attention_no:      0x07 0x81 0x81 0xC7 0xC7 0xFE 0xFF  (6 bytes)
		// attention_msg:     0x06 0x48 0x65 0x6C 0x6C 0x6F       ("Hello", 5 bytes)
		// attention_details: list of 1 TreeEntry
		//   tree entry = 3-element list (0x73)
		//     parameter_name:  0x03 0xAA 0xBB (2 bytes)
		//     parameter_value: 0x01 (absent)
		//     child_list:      0x01 (absent)
		input := []byte{
			0x74,                                           // list of 4
			0x05, 0x0A, 0x0B, 0x0C, 0x0D,                  // server_id
			0x07, 0x81, 0x81, 0xC7, 0xC7, 0xFE, 0xFF,      // attention_no
			0x06, 0x48, 0x65, 0x6C, 0x6C, 0x6F,            // attention_msg: "Hello"
			0x71,                                           // details: list of 1
			0x73,                                           // tree entry: list of 3
			0x03, 0xAA, 0xBB,                               // parameter_name
			0x01,                                           // parameter_value: absent
			0x01,                                           // child_list: absent
		}
		d := newDecoder(input)
		got := d.readAttentionResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil AttentionResponse")
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		wantAttNo := []byte{0x81, 0x81, 0xC7, 0xC7, 0xFE, 0xFF}
		if !bytes.Equal(got.AttentionNo, wantAttNo) {
			t.Fatalf("AttentionNo = %X, want %X", got.AttentionNo, wantAttNo)
		}

		if got.AttentionMsg == nil {
			t.Fatal("AttentionMsg = nil, want non-nil")
		}
		if *got.AttentionMsg != "Hello" {
			t.Fatalf("AttentionMsg = %q, want %q", *got.AttentionMsg, "Hello")
		}

		if len(got.AttentionDetails) != 1 {
			t.Fatalf("len(AttentionDetails) = %d, want 1", len(got.AttentionDetails))
		}
		wantName := []byte{0xAA, 0xBB}
		if !bytes.Equal(got.AttentionDetails[0].ParameterName, wantName) {
			t.Fatalf("AttentionDetails[0].ParameterName = %X, want %X", got.AttentionDetails[0].ParameterName, wantName)
		}
	})

	t.Run("optionals absent", func(t *testing.T) {
		// AttentionResponse = 4-element list (0x74)
		// server_id:         0x03 0xAA 0xBB  (2 bytes)
		// attention_no:      0x03 0xCC 0xDD  (2 bytes)
		// attention_msg:     0x01 (absent)
		// attention_details: 0x01 (absent)
		input := []byte{
			0x74,                   // list of 4
			0x03, 0xAA, 0xBB,      // server_id
			0x03, 0xCC, 0xDD,      // attention_no
			0x01,                   // attention_msg: absent
			0x01,                   // attention_details: absent
		}
		d := newDecoder(input)
		got := d.readAttentionResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil AttentionResponse")
		}

		wantServerID := []byte{0xAA, 0xBB}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		wantAttNo := []byte{0xCC, 0xDD}
		if !bytes.Equal(got.AttentionNo, wantAttNo) {
			t.Fatalf("AttentionNo = %X, want %X", got.AttentionNo, wantAttNo)
		}

		if got.AttentionMsg != nil {
			t.Fatalf("AttentionMsg = %v, want nil", *got.AttentionMsg)
		}
		if got.AttentionDetails != nil {
			t.Fatalf("AttentionDetails = %v, want nil", got.AttentionDetails)
		}
	})
}

func TestParseGetProcParameterResponse(t *testing.T) {
	t.Run("simple tree", func(t *testing.T) {
		// GetProcParameterResponse = 3-element list (0x73)
		// server_id:      0x05 0x0A 0x0B 0x0C 0x0D  (4 bytes)
		// tree_path:      list of 1 octet string
		//   0x71 → 0x07 0x01 0x00 0x01 0x08 0x00 0xFF  (OBIS 1-0:1.8.0*255)
		// parameter_tree: 3-element list (TreeEntry)
		//   parameter_name:  0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		//   parameter_value: 0x65 0x00 0x01 0xE2 0x40  (u32=123456)
		//   child_list:      0x01 (absent)
		input := []byte{
			0x73,                                           // list of 3
			0x05, 0x0A, 0x0B, 0x0C, 0x0D,                  // server_id
			0x71,                                           // tree_path: list of 1
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // path element
			0x73,                                           // parameter_tree: list of 3
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // parameter_name
			0x65, 0x00, 0x01, 0xE2, 0x40,                  // parameter_value: u32=123456
			0x01,                                           // child_list: absent
		}
		d := newDecoder(input)
		got := d.readGetProcParameterResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetProcParameterResponse")
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		if len(got.TreePath) != 1 {
			t.Fatalf("len(TreePath) = %d, want 1", len(got.TreePath))
		}
		wantPath := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(got.TreePath[0], wantPath) {
			t.Fatalf("TreePath[0] = %X, want %X", got.TreePath[0], wantPath)
		}

		if got.ParameterTree == nil {
			t.Fatal("ParameterTree = nil, want non-nil")
		}
		wantName := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(got.ParameterTree.ParameterName, wantName) {
			t.Fatalf("ParameterTree.ParameterName = %X, want %X", got.ParameterTree.ParameterName, wantName)
		}
		if got.ParameterTree.ParameterValue != Uint32(123456) {
			t.Fatalf("ParameterTree.ParameterValue = %v, want Uint32(123456)", got.ParameterTree.ParameterValue)
		}
	})
}

func TestParseGetProfileListResponse(t *testing.T) {
	t.Run("minimal with one period entry", func(t *testing.T) {
		// GetProfileListResponse = 9-element list (0x79)
		// server_id:            0x05 0x0A 0x0B 0x0C 0x0D  (4 bytes)
		// act_time:             0x01 (absent)
		// reg_period:           0x65 0x00 0x00 0x03 0x84  (u32=900)
		// parameter_tree_path:  list of 1 → 0x71 0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		// val_time:             0x72 0x62 0x01 0x65 0x00 0x00 0x30 0x39  (SecIndex=12345)
		// status:               0x01 (absent)
		// val_list:             list of 1 PeriodEntry
		//   PeriodEntry = 5-element list (0x75)
		//     obj_name:  0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		//     unit:      0x62 0x1E (u8=30)
		//     scaler:    0x52 0xFE (i8=-2)
		//     value:     0x65 0x00 0x01 0xE2 0x40 (u32=123456)
		//     signature: 0x01 (absent)
		// rawdata:              0x01 (absent)
		// period_signature:     0x01 (absent)
		input := []byte{
			0x79,                                           // list of 9
			0x05, 0x0A, 0x0B, 0x0C, 0x0D,                  // server_id
			0x01,                                           // act_time: absent
			0x65, 0x00, 0x00, 0x03, 0x84,                  // reg_period: u32=900
			0x71,                                           // parameter_tree_path: list of 1
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // path element
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39, // val_time: SecIndex=12345
			0x01,                                           // status: absent
			0x71,                                           // val_list: list of 1
			0x75,                                           // PeriodEntry: list of 5
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // obj_name
			0x62, 0x1E,                                     // unit: u8=30
			0x52, 0xFE,                                     // scaler: i8=-2
			0x65, 0x00, 0x01, 0xE2, 0x40,                  // value: u32=123456
			0x01,                                           // signature: absent
			0x01,                                           // rawdata: absent
			0x01,                                           // period_signature: absent
		}
		d := newDecoder(input)
		got := d.readGetProfileListResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetProfileListResponse")
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		if got.ActTime != nil {
			t.Fatalf("ActTime = %v, want nil", got.ActTime)
		}

		if got.RegPeriod == nil {
			t.Fatal("RegPeriod = nil, want non-nil")
		}
		if *got.RegPeriod != 900 {
			t.Fatalf("RegPeriod = %d, want 900", *got.RegPeriod)
		}

		if len(got.ParameterTreePath) != 1 {
			t.Fatalf("len(ParameterTreePath) = %d, want 1", len(got.ParameterTreePath))
		}

		if got.ValTime == nil {
			t.Fatal("ValTime = nil, want non-nil")
		}
		if got.ValTime.Tag != TimeSecIndex {
			t.Fatalf("ValTime.Tag = %d, want %d", got.ValTime.Tag, TimeSecIndex)
		}
		if got.ValTime.Value != 12345 {
			t.Fatalf("ValTime.Value = %d, want 12345", got.ValTime.Value)
		}

		if got.Status != nil {
			t.Fatalf("Status = %v, want nil", *got.Status)
		}

		if len(got.ValList) != 1 {
			t.Fatalf("len(ValList) = %d, want 1", len(got.ValList))
		}
		pe := got.ValList[0]
		wantObjName := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(pe.ObjName, wantObjName) {
			t.Fatalf("ValList[0].ObjName = %X, want %X", pe.ObjName, wantObjName)
		}
		if pe.Unit == nil || *pe.Unit != 30 {
			t.Fatalf("ValList[0].Unit = %v, want 30", pe.Unit)
		}
		if pe.Scaler == nil || *pe.Scaler != -2 {
			t.Fatalf("ValList[0].Scaler = %v, want -2", pe.Scaler)
		}
		if pe.Value != Uint32(123456) {
			t.Fatalf("ValList[0].Value = %v, want Uint32(123456)", pe.Value)
		}
		if pe.Signature != nil {
			t.Fatalf("ValList[0].Signature = %v, want nil", pe.Signature)
		}

		if got.RawData != nil {
			t.Fatalf("RawData = %v, want nil", got.RawData)
		}
		if got.PeriodSignature != nil {
			t.Fatalf("PeriodSignature = %v, want nil", got.PeriodSignature)
		}
	})

	t.Run("all optionals absent, empty val_list", func(t *testing.T) {
		// GetProfileListResponse = 9-element list (0x79)
		// All optional fields absent, empty val_list
		input := []byte{
			0x79,                               // list of 9
			0x03, 0x11, 0x22,                   // server_id: 2 bytes
			0x01,                               // act_time: absent
			0x01,                               // reg_period: absent
			0x71,                               // parameter_tree_path: list of 1
			0x03, 0xAA, 0xBB,                   // path element
			0x01,                               // val_time: absent
			0x01,                               // status: absent
			0x70,                               // val_list: list of 0
			0x01,                               // rawdata: absent
			0x01,                               // period_signature: absent
		}
		d := newDecoder(input)
		got := d.readGetProfileListResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetProfileListResponse")
		}

		wantServerID := []byte{0x11, 0x22}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}
		if got.ActTime != nil {
			t.Fatalf("ActTime = %v, want nil", got.ActTime)
		}
		if got.RegPeriod != nil {
			t.Fatalf("RegPeriod = %v, want nil", *got.RegPeriod)
		}
		if got.ValTime != nil {
			t.Fatalf("ValTime = %v, want nil", got.ValTime)
		}
		if got.Status != nil {
			t.Fatalf("Status = %v, want nil", *got.Status)
		}
		if len(got.ValList) != 0 {
			t.Fatalf("len(ValList) = %d, want 0", len(got.ValList))
		}
		if got.RawData != nil {
			t.Fatalf("RawData = %v, want nil", got.RawData)
		}
		if got.PeriodSignature != nil {
			t.Fatalf("PeriodSignature = %v, want nil", got.PeriodSignature)
		}
	})
}

func TestParseGetProfilePackResponse(t *testing.T) {
	t.Run("one header, one period with one value", func(t *testing.T) {
		// GetProfilePackResponse = 8-element list (0x78)
		// server_id:            0x05 0x0A 0x0B 0x0C 0x0D  (4 bytes)
		// act_time:             0x72 0x62 0x01 0x65 0x00 0x00 0x07 0xD0  (SecIndex=2000)
		// reg_period:           0x65 0x00 0x00 0x03 0x84  (u32=900)
		// parameter_tree_path:  list of 1 → 0x71 0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		// header_list:          list of 1 ProfileObjHeaderEntry
		//   ProfileObjHeaderEntry = 3-element list (0x73)
		//     obj_name: 0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		//     unit:     0x62 0x1E (u8=30)
		//     scaler:   0x52 0xFE (i8=-2)
		// period_list:          list of 1 ProfileObjPeriodEntry
		//   ProfileObjPeriodEntry = 3-element list (0x73)
		//     val_time:    0x72 0x62 0x01 0x65 0x00 0x00 0x30 0x39  (SecIndex=12345)
		//     status:      0x01 (absent)
		//     value_list:  list of 1 PeriodEntry
		//       PeriodEntry = 5-element list (0x75)
		//         obj_name:  0x07 0x01 0x00 0x01 0x08 0x00 0xFF
		//         unit:      0x62 0x1E (u8=30)
		//         scaler:    0x52 0xFE (i8=-2)
		//         value:     0x65 0x00 0x01 0xE2 0x40 (u32=123456)
		//         signature: 0x01 (absent)
		// rawdata:              0x01 (absent)
		// period_signature:     0x01 (absent)
		input := []byte{
			0x78,                                           // list of 8
			0x05, 0x0A, 0x0B, 0x0C, 0x0D,                  // server_id
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x07, 0xD0, // act_time: SecIndex=2000
			0x65, 0x00, 0x00, 0x03, 0x84,                  // reg_period: u32=900
			0x71,                                           // parameter_tree_path: list of 1
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // path element
			// header_list: list of 1
			0x71,
			0x73,                                           // ProfileObjHeaderEntry: list of 3
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // obj_name
			0x62, 0x1E,                                     // unit: u8=30
			0x52, 0xFE,                                     // scaler: i8=-2
			// period_list: list of 1
			0x71,
			0x73,                                           // ProfileObjPeriodEntry: list of 3
			0x72, 0x62, 0x01, 0x65, 0x00, 0x00, 0x30, 0x39, // val_time: SecIndex=12345
			0x01,                                           // status: absent
			// value_list: list of 1 PeriodEntry
			0x71,
			0x75,                                           // PeriodEntry: list of 5
			0x07, 0x01, 0x00, 0x01, 0x08, 0x00, 0xFF,      // obj_name
			0x62, 0x1E,                                     // unit: u8=30
			0x52, 0xFE,                                     // scaler: i8=-2
			0x65, 0x00, 0x01, 0xE2, 0x40,                  // value: u32=123456
			0x01,                                           // signature: absent
			// rawdata: absent
			0x01,
			// period_signature: absent
			0x01,
		}
		d := newDecoder(input)
		got := d.readGetProfilePackResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetProfilePackResponse")
		}

		wantServerID := []byte{0x0A, 0x0B, 0x0C, 0x0D}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}

		if got.ActTime == nil {
			t.Fatal("ActTime = nil, want non-nil")
		}
		if got.ActTime.Tag != TimeSecIndex || got.ActTime.Value != 2000 {
			t.Fatalf("ActTime = %+v, want SecIndex=2000", got.ActTime)
		}

		if got.RegPeriod == nil || *got.RegPeriod != 900 {
			t.Fatalf("RegPeriod = %v, want 900", got.RegPeriod)
		}

		if len(got.ParameterTreePath) != 1 {
			t.Fatalf("len(ParameterTreePath) = %d, want 1", len(got.ParameterTreePath))
		}

		// header_list
		if len(got.HeaderList) != 1 {
			t.Fatalf("len(HeaderList) = %d, want 1", len(got.HeaderList))
		}
		h := got.HeaderList[0]
		wantObjName := []byte{0x01, 0x00, 0x01, 0x08, 0x00, 0xFF}
		if !bytes.Equal(h.ObjName, wantObjName) {
			t.Fatalf("HeaderList[0].ObjName = %X, want %X", h.ObjName, wantObjName)
		}
		if h.Unit == nil || *h.Unit != 30 {
			t.Fatalf("HeaderList[0].Unit = %v, want 30", h.Unit)
		}
		if h.Scaler == nil || *h.Scaler != -2 {
			t.Fatalf("HeaderList[0].Scaler = %v, want -2", h.Scaler)
		}

		// period_list
		if len(got.PeriodList) != 1 {
			t.Fatalf("len(PeriodList) = %d, want 1", len(got.PeriodList))
		}
		p := got.PeriodList[0]
		if p.ValTime == nil || p.ValTime.Tag != TimeSecIndex || p.ValTime.Value != 12345 {
			t.Fatalf("PeriodList[0].ValTime = %+v, want SecIndex=12345", p.ValTime)
		}
		if p.Status != nil {
			t.Fatalf("PeriodList[0].Status = %v, want nil", *p.Status)
		}
		if len(p.ValueList) != 1 {
			t.Fatalf("len(PeriodList[0].ValueList) = %d, want 1", len(p.ValueList))
		}
		pe := p.ValueList[0]
		if !bytes.Equal(pe.ObjName, wantObjName) {
			t.Fatalf("PeriodList[0].ValueList[0].ObjName = %X, want %X", pe.ObjName, wantObjName)
		}
		if pe.Value != Uint32(123456) {
			t.Fatalf("PeriodList[0].ValueList[0].Value = %v, want Uint32(123456)", pe.Value)
		}

		if got.RawData != nil {
			t.Fatalf("RawData = %v, want nil", got.RawData)
		}
		if got.PeriodSignature != nil {
			t.Fatalf("PeriodSignature = %v, want nil", got.PeriodSignature)
		}
	})

	t.Run("all optionals absent, empty lists", func(t *testing.T) {
		input := []byte{
			0x78,                               // list of 8
			0x03, 0x11, 0x22,                   // server_id: 2 bytes
			0x01,                               // act_time: absent
			0x01,                               // reg_period: absent
			0x71,                               // parameter_tree_path: list of 1
			0x03, 0xAA, 0xBB,                   // path element
			0x70,                               // header_list: list of 0
			0x70,                               // period_list: list of 0
			0x01,                               // rawdata: absent
			0x01,                               // period_signature: absent
		}
		d := newDecoder(input)
		got := d.readGetProfilePackResponse()
		if d.err != nil {
			t.Fatalf("unexpected error: %v", d.err)
		}
		if got == nil {
			t.Fatal("got nil, want non-nil GetProfilePackResponse")
		}

		wantServerID := []byte{0x11, 0x22}
		if !bytes.Equal(got.ServerID, wantServerID) {
			t.Fatalf("ServerID = %X, want %X", got.ServerID, wantServerID)
		}
		if got.ActTime != nil {
			t.Fatalf("ActTime = %v, want nil", got.ActTime)
		}
		if got.RegPeriod != nil {
			t.Fatalf("RegPeriod = %v, want nil", *got.RegPeriod)
		}
		if len(got.HeaderList) != 0 {
			t.Fatalf("len(HeaderList) = %d, want 0", len(got.HeaderList))
		}
		if len(got.PeriodList) != 0 {
			t.Fatalf("len(PeriodList) = %d, want 0", len(got.PeriodList))
		}
		if got.RawData != nil {
			t.Fatalf("RawData = %v, want nil", got.RawData)
		}
		if got.PeriodSignature != nil {
			t.Fatalf("PeriodSignature = %v, want nil", got.PeriodSignature)
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
