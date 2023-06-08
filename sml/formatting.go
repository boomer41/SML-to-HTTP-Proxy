package sml

import "strings"

func prefixMultilineString(s string, prefix string) string {
	split := strings.Split(s, "\n")
	newString := ""

	for _, part := range split {
		newString += prefix + part + "\n"
	}

	if len(newString) == 0 {
		return ""
	}

	return newString[:len(newString)-1]
}
