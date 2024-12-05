package protocol

import (
	"strings"
)

func ParseCommand(input string) []string {
	return strings.Fields(input)
}
