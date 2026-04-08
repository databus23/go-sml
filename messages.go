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

// readAttentionResponse parses a 4-element SML list into an AttentionResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readAttentionResponse() *AttentionResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 4 {
		d.err = fmt.Errorf("sml: AttentionResponse list length = %d, want 4", n)
		return nil
	}

	serverID := d.readOctetString()
	attentionNo := d.readOctetString()
	attentionMsg := d.readOptionalStringPtr()

	var attentionDetails []TreeEntry
	if !d.isOptionalSkipped() {
		count := d.readListLength()
		if d.err != nil {
			return nil
		}
		attentionDetails = make([]TreeEntry, count)
		for i := range attentionDetails {
			attentionDetails[i] = d.readTreeEntry()
		}
	}

	if d.err != nil {
		return nil
	}
	return &AttentionResponse{
		ServerID:         serverID,
		AttentionNo:      attentionNo,
		AttentionMsg:     attentionMsg,
		AttentionDetails: attentionDetails,
	}
}

// readGetProcParameterResponse parses a 3-element SML list into a GetProcParameterResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readGetProcParameterResponse() *GetProcParameterResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 3 {
		d.err = fmt.Errorf("sml: GetProcParameterResponse list length = %d, want 3", n)
		return nil
	}

	serverID := d.readOctetString()
	treePath := d.readTreePath()
	entry := d.readTreeEntry()

	if d.err != nil {
		return nil
	}
	return &GetProcParameterResponse{
		ServerID:      serverID,
		TreePath:      treePath,
		ParameterTree: &entry,
	}
}

// readPeriodEntry parses a 5-element SML list into a PeriodEntry.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readPeriodEntry() PeriodEntry {
	n := d.readListLength()
	if d.err != nil {
		return PeriodEntry{}
	}
	if n != 5 {
		d.err = fmt.Errorf("sml: PeriodEntry list length = %d, want 5", n)
		return PeriodEntry{}
	}

	objName := d.readOctetString()
	unit := d.readOptionalUint8Ptr()
	scaler := d.readOptionalSignedPtr()
	value := d.readOptionalValue()
	signature := d.readOptionalOctetString()

	if d.err != nil {
		return PeriodEntry{}
	}
	return PeriodEntry{
		ObjName:   objName,
		Unit:      unit,
		Scaler:    scaler,
		Value:     value,
		Signature: signature,
	}
}

// readGetProfileListResponse parses a 9-element SML list into a GetProfileListResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readGetProfileListResponse() *GetProfileListResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 9 {
		d.err = fmt.Errorf("sml: GetProfileListResponse list length = %d, want 9", n)
		return nil
	}

	serverID := d.readOctetString()
	actTime := d.readOptionalTime()
	regPeriod := d.readOptionalUint32Ptr()
	parameterTreePath := d.readTreePath()
	valTime := d.readOptionalTime()
	status := d.readOptionalUnsignedPtr()

	valCount := d.readListLength()
	if d.err != nil {
		return nil
	}
	valList := make([]PeriodEntry, valCount)
	for i := range valList {
		valList[i] = d.readPeriodEntry()
	}

	rawData := d.readOptionalOctetString()
	periodSignature := d.readOptionalOctetString()

	if d.err != nil {
		return nil
	}
	return &GetProfileListResponse{
		ServerID:          serverID,
		ActTime:           actTime,
		RegPeriod:         regPeriod,
		ParameterTreePath: parameterTreePath,
		ValTime:           valTime,
		Status:            status,
		ValList:           valList,
		RawData:           rawData,
		PeriodSignature:   periodSignature,
	}
}

// readProfileObjHeaderEntry parses a 3-element SML list into a ProfileObjHeaderEntry.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readProfileObjHeaderEntry() ProfileObjHeaderEntry {
	n := d.readListLength()
	if d.err != nil {
		return ProfileObjHeaderEntry{}
	}
	if n != 3 {
		d.err = fmt.Errorf("sml: ProfileObjHeaderEntry list length = %d, want 3", n)
		return ProfileObjHeaderEntry{}
	}

	objName := d.readOctetString()
	unit := d.readOptionalUint8Ptr()
	scaler := d.readOptionalSignedPtr()

	if d.err != nil {
		return ProfileObjHeaderEntry{}
	}
	return ProfileObjHeaderEntry{
		ObjName: objName,
		Unit:    unit,
		Scaler:  scaler,
	}
}

// readProfileObjPeriodEntry parses a 3-element SML list into a ProfileObjPeriodEntry.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readProfileObjPeriodEntry() ProfileObjPeriodEntry {
	n := d.readListLength()
	if d.err != nil {
		return ProfileObjPeriodEntry{}
	}
	if n != 3 {
		d.err = fmt.Errorf("sml: ProfileObjPeriodEntry list length = %d, want 3", n)
		return ProfileObjPeriodEntry{}
	}

	valTime := d.readOptionalTime()
	status := d.readOptionalUnsignedPtr()

	valueCount := d.readListLength()
	if d.err != nil {
		return ProfileObjPeriodEntry{}
	}
	valueList := make([]PeriodEntry, valueCount)
	for i := range valueList {
		valueList[i] = d.readPeriodEntry()
	}

	if d.err != nil {
		return ProfileObjPeriodEntry{}
	}
	return ProfileObjPeriodEntry{
		ValTime:   valTime,
		Status:    status,
		ValueList: valueList,
	}
}

// readGetProfilePackResponse parses an 8-element SML list into a GetProfilePackResponse.
// The list header has not been read yet; this method reads and validates it.
func (d *decoder) readGetProfilePackResponse() *GetProfilePackResponse {
	n := d.readListLength()
	if d.err != nil {
		return nil
	}
	if n != 8 {
		d.err = fmt.Errorf("sml: GetProfilePackResponse list length = %d, want 8", n)
		return nil
	}

	serverID := d.readOctetString()
	actTime := d.readOptionalTime()
	regPeriod := d.readOptionalUint32Ptr()
	parameterTreePath := d.readTreePath()

	headerCount := d.readListLength()
	if d.err != nil {
		return nil
	}
	headerList := make([]ProfileObjHeaderEntry, headerCount)
	for i := range headerList {
		headerList[i] = d.readProfileObjHeaderEntry()
	}

	periodCount := d.readListLength()
	if d.err != nil {
		return nil
	}
	periodList := make([]ProfileObjPeriodEntry, periodCount)
	for i := range periodList {
		periodList[i] = d.readProfileObjPeriodEntry()
	}

	rawData := d.readOptionalOctetString()
	periodSignature := d.readOptionalOctetString()

	if d.err != nil {
		return nil
	}
	return &GetProfilePackResponse{
		ServerID:          serverID,
		ActTime:           actTime,
		RegPeriod:         regPeriod,
		ParameterTreePath: parameterTreePath,
		HeaderList:        headerList,
		PeriodList:        periodList,
		RawData:           rawData,
		PeriodSignature:   periodSignature,
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
