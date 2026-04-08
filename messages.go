package sml

import "fmt"

// readOpenResponse parses a 6-element SML list into an OpenResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readOpenResponse() *OpenResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 6 {
		d.err = fmt.Errorf("sml: OpenResponse list length = %d, want 6", n)
		return nil
	}

	codepage := d.readOptionalStringPtr()
	clientID := d.readOptionalOctetString()
	reqFileID := d.readOctetString()
	serverID := d.readOctetString()
	refTime := d.readOptionalTime()
	smlVersion := d.readOptionalUint8Ptr()

	if d.err != nil {
		return nil
	}
	return &OpenResponse{
		Codepage:   codepage,
		ClientID:   clientID,
		ReqFileID:  reqFileID,
		ServerID:   serverID,
		RefTime:    refTime,
		SmlVersion: smlVersion,
	}
}

// readCloseResponse parses a 1-element SML list into a CloseResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readCloseResponse() *CloseResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 1 {
		d.err = fmt.Errorf("sml: CloseResponse list length = %d, want 1", n)
		return nil
	}

	signature := d.readOptionalOctetString()

	if d.err != nil {
		return nil
	}
	return &CloseResponse{Signature: signature}
}
