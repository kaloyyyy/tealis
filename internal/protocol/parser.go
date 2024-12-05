package protocol

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseCommand(input string) []string {
	// Regular expression to match words or quoted strings (single or double quotes)
	re := regexp.MustCompile(`"([^"]*)"|'([^']*)'|[^\s]+`)
	fmt.Printf("input %s", input)
	matches := re.FindAllString(input, -1)
	var parts []string
	for _, match := range matches {
		// Remove the surrounding quotes if present
		trimmed := strings.Trim(match, `"'`)
		parts = append(parts, trimmed)
	}
	return parts
}
