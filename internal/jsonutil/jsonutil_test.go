package jsonutil

import "testing"

func TestGetStr(t *testing.T) {
	m := map[string]interface{}{
		"name":  "project",
		"count": 3,
	}
	if got := GetStr(m, "name"); got != "project" {
		t.Fatalf("GetStr string = %q, want project", got)
	}
	if got := GetStr(m, "count"); got != "" {
		t.Fatalf("GetStr non-string = %q, want empty", got)
	}
	if got := GetStr(m, "missing"); got != "" {
		t.Fatalf("GetStr missing = %q, want empty", got)
	}
}
