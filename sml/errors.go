package sml

import "fmt"

type InvalidMessage struct {
	error error
}

func (i InvalidMessage) Error() string {
	return fmt.Sprintf("invalid message: %v", i.error)
}

type InvalidFile struct {
	error error
}

func (i InvalidFile) Error() string {
	return fmt.Sprintf("invalid SML file: %v", i.error)
}
