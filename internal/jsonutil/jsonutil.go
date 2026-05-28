// Package jsonutil provides shared helpers for untyped JSON map traversal.
package jsonutil

func GetStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
