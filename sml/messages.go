package sml

import (
	"encoding/hex"
	"fmt"
)

type File struct {
	Messages []*Message
}

func (f *File) String() string {
	s := "File = {\n"
	s += " Messages = [\n"

	for _, m := range f.Messages {
		s += prefixMultilineString(m.String(), "  ") + "\n"
	}

	s += " ]\n"
	s += "}"

	return s
}

type Message struct {
	TransactionId []byte
	GroupNo       uint8
	AbortOnError  uint8
	MessageBody   MessageBody `sml:"choice:SML_MessageBody"`
	Crc16         uint16
	EndOfMessage  interface{}
}

func (m *Message) String() string {
	s := "Message = {\n"

	s += fmt.Sprintf(" TransactionId = %s\n", hex.EncodeToString(m.TransactionId))
	s += fmt.Sprintf(" GroupNo = %02x\n", m.GroupNo)
	s += fmt.Sprintf(" AbortOnError = %02x\n", m.AbortOnError)
	s += fmt.Sprintf(" MessageBody = {\n%s\n }\n", prefixMultilineString(m.MessageBody.String(), "  "))
	s += fmt.Sprintf(" Crc16 = %04x\n", m.Crc16)

	s += "}"
	return s
}

type MessageBody interface {
	fmt.Stringer
}

type PublicOpenResMessageBody struct {
	Codepage   []byte `sml:"optional"`
	ClientId   []byte `sml:"optional"`
	ReqFileId  []byte
	ServerId   []byte
	RefTime    interface{} `sml:"optional"`
	SmlVersion uint8       `sml:"optional"`
}

func (p *PublicOpenResMessageBody) String() string {
	s := "SML_PublicOpen.Res = {\n"

	s += fmt.Sprintf(" Codepage = %s\n", hex.EncodeToString(p.Codepage))
	s += fmt.Sprintf(" ClientId = %s\n", hex.EncodeToString(p.ClientId))
	s += fmt.Sprintf(" ReqFileId = %s\n", hex.EncodeToString(p.ReqFileId))
	s += fmt.Sprintf(" ServerId = %s\n", hex.EncodeToString(p.ServerId))
	s += fmt.Sprintf(" SmlVersion = %02x\n", p.SmlVersion)

	s += "}"
	return s
}

type PublicCloseResMessageBody struct {
	GlobalSignature []byte `sml:"optional"`
}

func (p *PublicCloseResMessageBody) String() string {
	s := "SML_PublicClose.Res = {\n"
	s += fmt.Sprintf(" GlobalSignature = %s\n", hex.EncodeToString(p.GlobalSignature))
	s += "}"
	return s
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

func (p *GetListResMessageBody) String() string {
	s := "SML_GetList.Res = {\n"

	s += fmt.Sprintf(" ClientId = %s\n", hex.EncodeToString(p.ClientId))
	s += fmt.Sprintf(" ServerId = %s\n", hex.EncodeToString(p.ServerId))
	s += fmt.Sprintf(" ListName = %s\n", hex.EncodeToString(p.ListName))

	s += " ValList = [\n"

	for _, v := range p.ValList {
		s += prefixMultilineString(v.String(), "  ") + "\n"
	}

	s += " ]\n"
	s += "}"
	return s
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

func (e *ListEntry) String() string {
	s := "ListEntry = {\n"

	obis, err := ObisToString(e.ObjName)

	if err != nil {
		obis = hex.EncodeToString(e.ObjName)
	}

	s += fmt.Sprintf(" ObjName = %s\n", obis)
	s += fmt.Sprintf(" Unit = %d\n", e.Unit)
	s += fmt.Sprintf(" Scaler = %d\n", e.Scaler)
	s += fmt.Sprintf(" Value = %s\n", e.stringValue())
	s += fmt.Sprintf(" ValueSignature = %s\n", hex.EncodeToString(e.ValueSignature))

	s += "}"
	return s
}

func (e *ListEntry) stringValue() string {
	if e.Value == nil {
		return "null"
	} else if v, ok := e.Value.(*string); ok {
		return "(string) " + *v
	} else if v, ok := e.Value.(*bool); ok {
		if *v {
			return "(bool) True"
		} else {
			return "(bool) False"
		}
	} else if v, ok := e.Value.([]byte); ok {
		return "(octet string) " + hex.EncodeToString(v)
	} else if v, ok := e.Value.(*uint8); ok {
		return "(uint8) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*uint16); ok {
		return "(uint16) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*uint32); ok {
		return "(uint32) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*uint64); ok {
		return "(uint64) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*int8); ok {
		return "(int8) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*int16); ok {
		return "(int16) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*int32); ok {
		return "(int32) " + fmt.Sprintf("%d", *v)
	} else if v, ok := e.Value.(*int64); ok {
		return "(int64) " + fmt.Sprintf("%d", *v)
	}

	return fmt.Sprintf("(unknown) %s", e.Value)
}
