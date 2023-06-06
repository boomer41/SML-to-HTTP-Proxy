package sml

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/sigurn/crc16"
)

type smlBinaryReader struct {
	reader          io.Reader
	ready           []byte
	escapingToCheck []byte

	crcTable      *crc16.Table
	doCrc         bool
	crc           uint16
	crcDataLength int
}

func newSmlBinaryReader(r io.Reader) *smlBinaryReader {
	return &smlBinaryReader{
		reader:          r,
		ready:           make([]byte, 0),
		escapingToCheck: make([]byte, 0),
		crcTable:        crc16.MakeTable(crc16.CRC16_X_25),
		doCrc:           false,
		crc:             0,
	}
}

var foundBeginOfMessage = errors.New("found begin of message in stream")
var escapeError = &InvalidMessage{
	error: errors.New("escape error"),
}

type foundEndOfMessage struct {
	countPaddingBytes  int
	crcDataLength      int
	expectedCheckSum   uint16
	calculatedChecksum uint16
}

func (f *foundEndOfMessage) Error() string {
	return "found end of message in stream"
}

func (r *smlBinaryReader) readBuffer(wantedLength int) (data []byte, e error) {
	if len(r.ready) >= wantedLength {
		data = r.ready[0:wantedLength]
		r.ready = r.ready[wantedLength:]
		return
	}

	forceRead := false

	for {
		// When we still have something to check for, do it first.
		// We may already be EOF and the end of message marker needs to be returned.
		// But only do it ONCE, because we might need more data for escape checking
		if forceRead || len(r.escapingToCheck) == 0 {
			tmp := make([]byte, 32)
			n, err := r.reader.Read(tmp)

			if err != nil {
				e = err
				return
			}

			tmp = tmp[0:n]

			r.escapingToCheck = append(r.escapingToCheck, tmp...)
		}

		forceRead = true

		for {
			// When we have enough, return it immediately
			// This way we can actually read everything before receiving a "found EOF marker" error
			if len(r.ready) >= wantedLength {
				data = r.ready[0:wantedLength]
				r.ready = r.ready[wantedLength:]
				return
			}

			escapeOffset := bytes.IndexByte(r.escapingToCheck, 0x1b)

			if escapeOffset < 0 {
				r.ready = append(r.ready, r.escapingToCheck...)
				r.appendCrc(r.escapingToCheck)
				r.escapingToCheck = make([]byte, 0)
				break
			}

			if escapeOffset > 0 {
				r.ready = append(r.ready, r.escapingToCheck[0:escapeOffset]...)
				r.appendCrc(r.escapingToCheck[0:escapeOffset])
				r.escapingToCheck = r.escapingToCheck[escapeOffset:]
				continue
			}

			// If we don't have 7 following bytes to check, loop around
			if len(r.escapingToCheck) < 8 {
				break
			}

			// Handle escaping
			if r.escapingToCheck[1] == 0x1b && r.escapingToCheck[2] == 0x1b && r.escapingToCheck[3] == 0x1b {
				escapeData := r.escapingToCheck[4:8]
				r.escapingToCheck = r.escapingToCheck[8:]

				// Encoded 0x1b1b1b1b
				if bytes.Compare(escapeData, []byte{0x1b, 0x1b, 0x1b, 0x1b}) == 0 {
					r.ready = append(r.ready, escapeData...)
					r.appendCrc([]byte{0x1b, 0x1b, 0x1b, 0x1b, 0x1b, 0x1b, 0x1b, 0x1b})
				} else
				// Encoded begin of message
				if bytes.Compare(escapeData, []byte{0x01, 0x01, 0x01, 0x01}) == 0 {
					r.ready = make([]byte, 0)

					r.crc = crc16.Init(r.crcTable)
					r.crcDataLength = 0
					r.doCrc = true

					r.appendCrc([]byte{
						0x1b, 0x1b, 0x1b, 0x1b,
						0x01, 0x01, 0x01, 0x01,
					})
					return nil, foundBeginOfMessage
				} else
				// End of message
				if escapeData[0] == 0x1a {
					countPaddingBytes := escapeData[1]

					if countPaddingBytes > 3 {
						return nil, escapeError
					}

					// Do not add the CRC bytes
					r.appendCrc([]byte{0x1b, 0x1b, 0x1b, 0x1b})
					r.appendCrc(escapeData[0:2])

					checksum := crc16.Complete(r.crc, r.crcTable)
					checksum = checksum&0x00FF<<8 | checksum&0xFF00>>8

					return nil, &foundEndOfMessage{
						countPaddingBytes:  int(countPaddingBytes),
						expectedCheckSum:   uint16(escapeData[2])<<8 | uint16(escapeData[3]),
						crcDataLength:      r.crcDataLength,
						calculatedChecksum: checksum,
					}
				} else {
					return nil, escapeError
				}
			} else {
				r.ready = append(r.ready, r.escapingToCheck[0])
				r.appendCrc(r.escapingToCheck[0:1])
				r.escapingToCheck = r.escapingToCheck[1:]
			}
		}
	}
}

func (r *smlBinaryReader) appendCrc(data []byte) {
	if !r.doCrc {
		return
	}

	r.crc = crc16.Update(r.crc, data, r.crcTable)
	r.crcDataLength = r.crcDataLength + len(data)
}

func (r *smlBinaryReader) readTypeLength() (tlf binaryTypeLengthField, e error) {
	firstTlvByte, err := r.readBuffer(1)

	if err != nil {
		e = err
		return
	}

	typeId := firstTlvByte[0] & 0x70 >> 4
	dataLength := firstTlvByte[0] & 0x0F
	moreBytesFollowing := (firstTlvByte[0] & 0x80) != 0

	if moreBytesFollowing {
		nextByte, err := r.readBuffer(1)

		if err != nil {
			e = err
			return
		}

		moreBytesFollowing = (nextByte[0] & 0x80) != 0

		if moreBytesFollowing {
			e = &InvalidMessage{
				error: errors.New("only SML type-length-fields with up to two bytes are supported"),
			}
			return
		}

		if nextByteMode := (nextByte[0] & 0x70) >> 4; nextByteMode != 0 {
			e = &InvalidMessage{
				error: fmt.Errorf("unknown mode %1x for second SML tlv byte", nextByteMode),
			}
			return
		}

		dataLength = (dataLength << 4) | nextByte[0]&0x0F
	}

	tlf = binaryTypeLengthField{
		typeId,
		int(dataLength),
	}
	e = nil
	return
}

func (r *smlBinaryReader) readToken() (smlToken, error) {
	tlf, err := r.readTypeLength()

	if err != nil {
		return nil, err
	}

	if tlf.dataLength == 0 && tlf.dataType == 0 {
		return &smlEndOfMessage{}, nil
	}

	switch tlf.dataType {
	case 0x0:
		return r.readOctetString(&tlf)
	case 0x4:
		return r.readBoolean(&tlf)
	case 0x5, 0x6:
		return r.readNumber(&tlf)
	case 0x7:
		return r.readList(&tlf)
	default:
		return nil, &InvalidMessage{
			error: fmt.Errorf("unknown SML type %1x", tlf.dataType),
		}
	}
}

func (r *smlBinaryReader) readOctetString(tlf *binaryTypeLengthField) (smlToken, error) {
	if tlf.dataLength == 0 {
		return nil, &InvalidMessage{
			error: fmt.Errorf("invalid data length value %d for octet string", tlf.dataLength),
		}
	}

	data, err := r.readBuffer(tlf.dataLength - 1)

	if err != nil {
		return nil, err
	}

	return &smlOctetString{
		value: data,
	}, nil
}

func (r *smlBinaryReader) readBoolean(tlf *binaryTypeLengthField) (smlToken, error) {
	if tlf.dataLength != 2 {
		return nil, &InvalidMessage{
			error: fmt.Errorf("invalid data length %d for SML boolean", tlf.dataLength),
		}
	}

	data, err := r.readBuffer(1)

	if err != nil {
		return nil, err
	}

	return &smlBoolean{
		value: data[0] != 0x00,
	}, nil
}

func (r *smlBinaryReader) readNumber(tlf *binaryTypeLengthField) (smlToken, error) {
	realDataLength := tlf.dataLength - 1

	data, err := r.readBuffer(realDataLength)

	if err != nil {
		return nil, err
	}

	// Fill up bytes...
	if realDataLength == 3 {
		realDataLength = 4
	} else if realDataLength >= 5 && realDataLength <= 7 {
		realDataLength = 8
	}

	for len(data) != realDataLength {
		data = append([]byte{0}, data...)
	}

	dataReader := bytes.NewReader(data)
	var token smlToken

	switch tlf.dataType {
	// Signed
	case 0x5:
		switch realDataLength {
		case 1:
			var v smlSigned8
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 2:
			var v smlSigned16
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 4:
			var v smlSigned32
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 8:
			var v smlSigned64
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		}
	// Unsigned
	case 0x6:
		switch realDataLength {
		case 1:
			var v smlUnsigned8
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 2:
			var v smlUnsigned16
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 4:
			var v smlUnsigned32
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		case 8:
			var v smlUnsigned64
			err = binary.Read(dataReader, binary.BigEndian, &v.value)
			token = &v
		}
	}

	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, &InvalidMessage{
			error: fmt.Errorf("unsupported numeric SML type with type %1x and length %d", tlf.dataType, tlf.dataLength),
		}
	}

	return token, nil
}

func (r *smlBinaryReader) readList(tlf *binaryTypeLengthField) (smlToken, error) {
	elementCount := tlf.dataLength

	tokens := make([]smlToken, elementCount)

	for i := 0; i < elementCount; i++ {
		token, err := r.readToken()

		if err != nil {
			return nil, err
		}

		tokens[i] = token
	}

	return &smlList{
		value: tokens,
	}, nil
}

func (r *smlBinaryReader) readMessageBundleWithoutRetry() (*unparsedMessageBundle, error) {
	r.doCrc = false
	r.crc = 0
	r.crcDataLength = 0

	for {
		_, err := r.readBuffer(1)

		if err == foundBeginOfMessage {
			break
		}

		if err == nil {
			continue
		}

		return nil, err
	}

	message := &unparsedMessageBundle{
		messages: make([]*smlList, 0),
	}

	var endOfMessageMarker *foundEndOfMessage

	endOfMessageCount := 0

	for {
		tok, err := r.readToken()

		if err != nil {
			if marker, ok := err.(*foundEndOfMessage); ok {
				endOfMessageMarker = marker
				break
			}

			return nil, err
		}

		_, isEndOfMessage := tok.(*smlEndOfMessage)

		if isEndOfMessage {
			endOfMessageCount = endOfMessageCount + 1

			if endOfMessageCount > 3 {
				// 0 to 3 zero-bytes may come
				return nil, &InvalidMessage{
					error: errors.New("excessive padding bytes found"),
				}
			}

			continue
		}

		if endOfMessageCount > 0 {
			return nil, &InvalidMessage{
				error: errors.New("unexpected data after end of message marker"),
			}
		}

		list, ok := tok.(*smlList)

		if !ok {
			return nil, &InvalidMessage{
				error: fmt.Errorf("expected SML list, but got %v", tok),
			}
		}

		message.messages = append(message.messages, list)
	}

	if (endOfMessageMarker.crcDataLength-2)%4 != 0 {
		return nil, &InvalidMessage{
			error: fmt.Errorf("data must be divisible by 4"),
		}
	}

	if endOfMessageMarker.countPaddingBytes != endOfMessageCount {
		return nil, &InvalidMessage{
			error: fmt.Errorf("expected %d padding bytes, found %d", endOfMessageMarker.countPaddingBytes, endOfMessageCount),
		}
	}

	if endOfMessageMarker.calculatedChecksum != endOfMessageMarker.expectedCheckSum {
		return nil, &InvalidMessage{
			error: fmt.Errorf("crc error: expected %02x, calculated %02x", endOfMessageMarker.expectedCheckSum, endOfMessageMarker.calculatedChecksum),
		}
	}

	return message, nil
}

func (r *smlBinaryReader) readMessageBundle() (*unparsedMessageBundle, error) {
	for {
		msg, err := r.readMessageBundleWithoutRetry()

		if err == nil {
			return msg, nil
		}

		if _, ok := err.(*foundEndOfMessage); ok {
			continue
		}

		if _, ok := err.(*InvalidMessage); ok {
			continue
		}

		return nil, err
	}
}
