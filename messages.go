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

// readListEntry parses a 7-element SML list into a ListEntry.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readListEntry() ListEntry {
	n := d.readListLength()
	if d.err != nil {
		return ListEntry{}
	}
	if n != 7 {
		d.err = fmt.Errorf("sml: ListEntry list length = %d, want 7", n)
		return ListEntry{}
	}

	objName := d.readOctetString()
	status := d.readOptionalUnsignedPtr()
	valTime := d.readOptionalTime()
	unit := d.readOptionalUint8Ptr()
	scaler := d.readOptionalSignedPtr()
	value := d.readOptionalValue()
	signature := d.readOptionalOctetString()

	if d.err != nil {
		return ListEntry{}
	}
	return ListEntry{
		ObjName:   objName,
		Status:    status,
		ValTime:   valTime,
		Unit:      unit,
		Scaler:    scaler,
		Value:     value,
		Signature: signature,
	}
}

// readGetListResponse parses a 7-element SML list into a GetListResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readGetListResponse() *GetListResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 7 {
		d.err = fmt.Errorf("sml: GetListResponse list length = %d, want 7", n)
		return nil
	}

	clientID := d.readOptionalOctetString()
	serverID := d.readOctetString()
	listName := d.readOptionalOctetString()
	actSensorTime := d.readOptionalTime()

	valCount := d.readListLength()
	if d.err != nil {
		return nil
	}
	valList := make([]ListEntry, valCount)
	for i := range valList {
		valList[i] = d.readListEntry()
	}

	signature := d.readOptionalOctetString()
	actGatewayTime := d.readOptionalTime()

	if d.err != nil {
		return nil
	}
	return &GetListResponse{
		ClientID:       clientID,
		ServerID:       serverID,
		ListName:       listName,
		ActSensorTime:  actSensorTime,
		ValList:        valList,
		Signature:      signature,
		ActGatewayTime: actGatewayTime,
	}
}
