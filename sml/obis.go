package sml

import (
	"errors"
	"fmt"
)

func ObisToString(val []byte) (string, error) {
	if len(val) != 6 {
		return "", errors.New("OBIS values must consist of 6 bytes")
	}

	return fmt.Sprintf("%d-%d:%d.%d.%d*%d", val[0], val[1], val[2], val[3], val[4], val[5]), nil
}
