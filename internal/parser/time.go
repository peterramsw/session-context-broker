package parser

import "strings"

// FormatTimestamp converts an ISO timestamp string to "MM-DD HH:MM" format.
func FormatTimestamp(tsStr string) string {
	if tsStr == "" {
		return "??-?? ??:??"
	}
	tsStr = strings.Replace(tsStr, "Z", "+00:00", 1)
	t, err := parseISO(tsStr)
	if err != nil {
		return "??-?? ??:??"
	}
	return t.Format("01-02 15:04")
}
