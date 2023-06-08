package sml

import "encoding/json"

type File struct {
	Messages []*Message
}

func (f *File) String() string {
	v, _ := json.Marshal(f)

	return string(v)
}

func (f *File) StringPretty() string {
	v, _ := json.MarshalIndent(f, "", "  ")

	return string(v)
}

type Message struct {
	TransactionId []byte
	GroupNo       uint8
	AbortOnError  uint8
	MessageBody   MessageBody `sml:"choice:SML_MessageBody"`
	Crc16         uint16
	EndOfMessage  interface{}
}

type MessageBody interface {
}

type PublicOpenResMessageBody struct {
	Codepage   []byte `sml:"optional"`
	ClientId   []byte `sml:"optional"`
	ReqFileId  []byte
	ServerId   []byte
	RefTime    interface{} `sml:"optional"`
	SmlVersion uint8       `sml:"optional"`
}

type PublicCloseResMessageBody struct {
	GlobalSignature []byte `sml:"optional"`
}

type GetListResMessageBody struct {
	ClientId       []byte `sml:"optional"`
	ServerId       []byte
	ListName       []byte      `sml:"optional"`
	ActSensorTime  interface{} `sml:"optional"`
	ValList        []*ListEntry
	ListSignature  []byte      `sml:"optional"`
	ActGatewayTime interface{} `sml:"optional"`
}

type ListEntry struct {
	ObjName        []byte
	Status         interface{} `sml:"implicit_choice:uint8:uint16:uint32:uint64,optional"`
	ValTime        interface{} `sml:"optional"`
	Unit           uint8       `sml:"optional"`
	Scaler         int8        `sml:"optional"`
	Value          interface{} `sml:"implicit_choice:bool:octet_string:int8:int16:int32:int64:uint8:uint16:uint32:uint64"`
	ValueSignature []byte      `sml:"optional"`
}
