package strutil

import "strings"

func PowershellEscape(s string) string {
	fixedString := strings.ReplaceAll(s, "\"", "`\"")
	fixedString = strings.ReplaceAll(fixedString, "\n", "`n")
	fixedString = strings.ReplaceAll(fixedString, "\r", "`r")
	fixedString = strings.ReplaceAll(fixedString, "\t", "`t")
	return fixedString
}
