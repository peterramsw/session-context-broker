package antigravitycodec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

type transcriptStep struct {
	StepIndex       int             `json:"step_index"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	Status          string          `json:"status"`
	CreatedAt       string          `json:"created_at"`
	Content         string          `json:"content"`
	Error           string          `json:"error"`
	ToolCalls       []toolCall      `json:"tool_calls"`
	Thinking        json.RawMessage `json:"thinking"`
	TruncatedFields []string        `json:"truncated_fields"`
}

type toolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

var toolResultNames = map[string]string{
	"RUN_COMMAND":     "run_command",
	"VIEW_FILE":       "view_file",
	"LIST_DIRECTORY":  "list_dir",
	"MCP_TOOL":        "call_mcp_tool",
	"GREP_SEARCH":     "grep_search",
	"CODE_ACTION":     "code_action",
	"INVOKE_SUBAGENT": "invoke_subagent",
	"SEARCH_WEB":      "search_web",
	"GENERIC":         "generic",
	"ERROR_MESSAGE":   "error_message",
	"CHECKPOINT":      "checkpoint",
	"ASK_PERMISSION":  "ask_permission",
	"MANAGE_TASK":     "manage_task",
	"SCHEDULE":        "schedule",
	"WRITE_TO_FILE":   "write_to_file",
	"REPLACE_CONTENT": "replace_file_content",
	"MULTI_REPLACE":   "multi_replace_file_content",
}

func parseStep(base session.SessionEvent, step transcriptStep) []session.SessionEvent {
	base.Timestamp = step.CreatedAt
	base.Metadata["step_index"] = step.StepIndex
	base.Metadata["source"] = step.Source
	base.Metadata["status"] = step.Status
	base.Metadata["antigravity_type"] = step.Type
	if len(step.TruncatedFields) > 0 {
		base.Metadata["truncated_fields"] = step.TruncatedFields
	}

	var events []session.SessionEvent
	switch step.Type {
	case "USER_INPUT":
		if step.Source == "USER_EXPLICIT" {
			base.EventType = "message"
			base.Role = "user"
			base.Content = firstNonEmpty(extractTagged(step.Content, "USER_REQUEST"), step.Content)
			return []session.SessionEvent{base}
		}
		base.EventType = "noise"
		base.Content = step.Content
		return []session.SessionEvent{base}

	case "PLANNER_RESPONSE":
		if strings.TrimSpace(step.Content) != "" {
			msg := base
			msg.EventType = "message"
			msg.Role = "assistant"
			msg.Content = step.Content
			events = append(events, msg)
		}
		if thinking := thinkingText(step.Thinking); strings.TrimSpace(thinking) != "" {
			reasoning := base
			reasoning.EventType = "reasoning"
			reasoning.Content = thinking
			events = append(events, reasoning)
		}
		for i, call := range step.ToolCalls {
			tool := base
			tool.EventType = "tool_call"
			tool.Tool = &session.SessionTool{
				CallID:    fmt.Sprintf("agy-step-%d-call-%d", step.StepIndex, i+1),
				Name:      firstNonEmpty(call.Name, "tool_call"),
				Arguments: compactJSON(call.Args),
				Status:    step.Status,
			}
			tool.Content = tool.Tool.Arguments
			events = append(events, tool)
		}
		if len(events) == 0 {
			base.EventType = "noise"
			base.Content = step.Content
			events = append(events, base)
		}
		return events
	}

	if name, ok := toolResultNames[step.Type]; ok || step.Source == "MODEL" {
		if !ok {
			name = strings.ToLower(step.Type)
		}
		content := firstNonEmpty(step.Error, step.Content)
		base.EventType = "tool_result"
		base.Tool = &session.SessionTool{
			CallID: fmt.Sprintf("agy-step-%d-result", step.StepIndex),
			Name:   name,
			Result: content,
			Status: statusText(step.Type, step.Status, content),
		}
		base.Content = content
		return []session.SessionEvent{base}
	}

	base.EventType = "noise"
	base.Content = firstNonEmpty(step.Error, step.Content)
	return []session.SessionEvent{base}
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return session.MarshalNoEscape(v)
}

func thinkingText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return compactJSON(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, format := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(format, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

func truncateLine(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

var failurePattern = regexp.MustCompile(`(?i)(encountered error in step execution|the command failed|exit code:\s*[1-9][0-9]*|resource_exhausted)`)

func statusText(stepType string, status string, content string) string {
	if status == "ERROR" {
		return "FAILED"
	}
	if stepType == "ERROR_MESSAGE" {
		return "FAILED"
	}
	if failurePattern.MatchString(content) {
		return "FAILED"
	}
	return "ok"
}

func extractTagged(content, tag string) string {
	re := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(tag) + `>\s*(.*?)\s*</` + regexp.QuoteMeta(tag) + `>`)
	if m := re.FindStringSubmatch(content); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func stringArg(raw json.RawMessage, keys ...string) string {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := obj[key]; ok {
			switch typed := value.(type) {
			case string:
				return unquoteNested(typed)
			case float64:
				return strconv.FormatFloat(typed, 'f', -1, 64)
			}
		}
	}
	return ""
}

func unquoteNested(value string) string {
	value = strings.TrimSpace(value)
	var decoded string
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded
	}
	return value
}
