package sml

type unparsedMessageBundle struct {
	messages []*smlList
}

type binaryTypeLengthField struct {
	dataType   uint8
	dataLength int
}

type smlToken interface {
}

type smlOctetString struct {
	value []byte
}

type smlBoolean struct {
	value bool
}

type smlUnsigned8 struct {
	value uint8
}

type smlUnsigned16 struct {
	value uint16
}

type smlUnsigned32 struct {
	value uint32
}

type smlUnsigned64 struct {
	value uint64
}

type smlSigned8 struct {
	value int8
}

type smlSigned16 struct {
	value int16
}

type smlSigned32 struct {
	value int32
}

type smlSigned64 struct {
	value int64
}

type smlList struct {
	value []smlToken
}

type smlEndOfMessage struct {
}
