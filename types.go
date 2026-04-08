package sml

// File is a decoded SML transmission containing one or more messages.
type File struct {
	Messages []*Message
}

// Message is a single SML message within a file.
type Message struct {
	TransactionID []byte
	GroupID       uint8
	AbortOnError  uint8
	Body          MessageBody
	CRC           uint16
}

// MessageBody is implemented by each specific message type.
type MessageBody interface {
	messageTag() uint32
}

// Value is implemented by all SML value types.
type Value interface {
	smlValue()
}

// Concrete Value types — named types preserve wire encoding size.
type OctetString []byte
type Bool bool
type Int8 int8
type Int16 int16
type Int32 int32
type Int64 int64
type Uint8 uint8
type Uint16 uint16
type Uint32 uint32
type Uint64 uint64

func (OctetString) smlValue() {}
func (Bool) smlValue()        {}
func (Int8) smlValue()        {}
func (Int16) smlValue()       {}
func (Int32) smlValue()       {}
func (Int64) smlValue()       {}
func (Uint8) smlValue()       {}
func (Uint16) smlValue()      {}
func (Uint32) smlValue()      {}
func (Uint64) smlValue()      {}

// Time represents a tagged SML time value.
type Time struct {
	Tag   uint8  // TimeSecIndex or TimeTimestamp
	Value uint32
}

const (
	TimeSecIndex  uint8 = 0x01
	TimeTimestamp uint8 = 0x02
)

// TreePath is a flat path of OBIS code segments.
type TreePath [][]byte

// TreeEntry is a recursive parameter tree node.
type TreeEntry struct {
	ParameterName  []byte
	ParameterValue Value
	Children       []TreeEntry
}

// ListEntry represents a single meter reading.
type ListEntry struct {
	ObjName   []byte // OBIS code (typically 6 bytes)
	Status    *uint64
	ValTime   *Time
	Unit      *uint8 // DLMS unit code
	Scaler    *int8
	Value     Value
	Signature []byte
}

// PeriodEntry is used in GetProfileListResponse.
type PeriodEntry struct {
	ObjName   []byte
	Unit      *uint8
	Scaler    *int8
	Value     Value
	Signature []byte
}

// ProfileObjHeaderEntry is used in GetProfilePackResponse.
type ProfileObjHeaderEntry struct {
	ObjName []byte
	Unit    *uint8
	Scaler  *int8
}

// ProfileObjPeriodEntry is used in GetProfilePackResponse.
type ProfileObjPeriodEntry struct {
	ValTime   *Time
	Status    *uint64
	ValueList []PeriodEntry
}

// Message tag constants for SML message body types.
const (
	tagOpenResponse             uint32 = 0x00000101
	tagCloseResponse            uint32 = 0x0000020b
	tagGetProfilePackResponse   uint32 = 0x00000301
	tagGetProfileListResponse   uint32 = 0x00000701
	tagGetProcParameterResponse uint32 = 0x00000501
	tagGetListResponse          uint32 = 0x00000701
	tagAttentionResponse        uint32 = 0x0000ff01
)

// OpenResponse is the body of an SML open response message.
type OpenResponse struct {
	Codepage   *string
	ClientID   []byte
	ReqFileID  []byte
	ServerID   []byte
	RefTime    *Time
	SmlVersion *uint8
}

// CloseResponse is the body of an SML close response message.
type CloseResponse struct {
	Signature []byte
}

// GetListResponse is the body of an SML get-list response message.
type GetListResponse struct {
	ClientID       []byte
	ServerID       []byte
	ListName       []byte
	ActSensorTime  *Time
	ValList        []ListEntry
	Signature      []byte
	ActGatewayTime *Time
}

// AttentionResponse is the body of an SML attention response message.
type AttentionResponse struct {
	ServerID         []byte
	AttentionNo      []byte
	AttentionMsg     *string
	AttentionDetails []TreeEntry
}

// GetProcParameterResponse is the body of an SML get-proc-parameter response message.
type GetProcParameterResponse struct {
	ServerID      []byte
	TreePath      TreePath
	ParameterTree *TreeEntry
}

// GetProfileListResponse is the body of an SML get-profile-list response message.
type GetProfileListResponse struct {
	ServerID          []byte
	ActTime           *Time
	RegPeriod         *uint32
	ParameterTreePath TreePath
	ValTime           *Time
	Status            *uint64
	ValList           []PeriodEntry
	RawData           []byte
	PeriodSignature   []byte
}

// GetProfilePackResponse is the body of an SML get-profile-pack response message.
type GetProfilePackResponse struct {
	ServerID          []byte
	ActTime           *Time
	RegPeriod         *uint32
	ParameterTreePath TreePath
	HeaderList        []ProfileObjHeaderEntry
	PeriodList        []ProfileObjPeriodEntry
	RawData           []byte
	PeriodSignature   []byte
}

func (*OpenResponse) messageTag() uint32             { return tagOpenResponse }
func (*CloseResponse) messageTag() uint32            { return tagCloseResponse }
func (*GetListResponse) messageTag() uint32          { return tagGetListResponse }
func (*AttentionResponse) messageTag() uint32        { return tagAttentionResponse }
func (*GetProcParameterResponse) messageTag() uint32 { return tagGetProcParameterResponse }
func (*GetProfileListResponse) messageTag() uint32   { return tagGetProfileListResponse }
func (*GetProfilePackResponse) messageTag() uint32   { return tagGetProfilePackResponse }

// DecodeOptions controls decoder behavior.
type DecodeOptions struct {
	Strict bool // default: true — fail on first malformed message
}
