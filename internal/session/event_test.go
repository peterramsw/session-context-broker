package session

import "testing"

func TestToolInputMarshalNoEscape(t *testing.T) {
	input := ToolInput{Raw: map[string]any{
		"html": "<tag>",
	}}
	got := input.MarshalNoEscape()
	want := `{"html":"<tag>"}`
	if got != want {
		t.Fatalf("MarshalNoEscape() = %q, want %q", got, want)
	}
}

func TestToolInputMarshalNoEscape_NilRaw(t *testing.T) {
	got := (ToolInput{}).MarshalNoEscape()
	if got != "{}" {
		t.Fatalf("MarshalNoEscape() = %q, want {}", got)
	}
}

func TestToolResultStatus(t *testing.T) {
	tests := []struct {
		name   string
		result ToolResult
		want   string
	}{
		{name: "success", result: ToolResult{Success: true}, want: "ok"},
		{name: "failure", result: ToolResult{Success: false}, want: "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Status(); got != tt.want {
				t.Fatalf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToolResultSummary(t *testing.T) {
	tests := []struct {
		name   string
		result ToolResult
		want   string
	}{
		{name: "success with text", result: ToolResult{Success: true, Text: "first\nsecond"}, want: " -> ok: first"},
		{name: "failure with text", result: ToolResult{Success: false, Text: "bad"}, want: " -> FAILED: bad"},
		{name: "success without text", result: ToolResult{Success: true}, want: " -> ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Summary(); got != tt.want {
				t.Fatalf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}
