package sml

import "io"

type Reader interface {
	ReadFile() (*File, error)
}

type smlReaderImpl struct {
	binary *smlBinaryReader
}

func NewReader(reader io.Reader) Reader {
	return &smlReaderImpl{
		binary: newSmlBinaryReader(reader),
	}
}

func (s *smlReaderImpl) ReadFile() (*File, error) {
	unparsed, err := s.binary.readMessageBundle()

	if err != nil {
		return nil, err
	}

	return deserializeMessageBundle(unparsed)
}
