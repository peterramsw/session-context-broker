package codexcodec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

type envelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionMetaPayload struct {
	SessionID     string `json:"session_id"`
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	Originator    string `json:"originator"`
	CLIVersion    string `json:"cli_version"`
	ModelProvider string `json:"model_provider"`
}

type typedPayload struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Arguments json.RawMessage `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	CallID    string          `json:"call_id"`
	Output    json.RawMessage `json:"output"`
	Status    string          `json:"status"`
	Summary   json.RawMessage `json:"summary"`
	Message   string          `json:"message"`
	Duration  json.RawMessage `json:"duration"`
	Result    json.RawMessage `json:"result"`
	Tools     json.RawMessage `json:"tools"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func parseResponseItem(base session.SessionEvent, raw json.RawMessage) session.SessionEvent {
	var payload typedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		base.EventType = "unknown"
		base.Content = compactJSON(raw)
		base.Metadata["parse_warning"] = err.Error()
		return base
	}
	base.Metadata["response_item_type"] = payload.Type

	switch payload.Type {
	case "message":
		base.EventType = "message"
		base.Role = payload.Role
		base.Content = contentText(payload.Content)
		return base
	case "function_call", "custom_tool_call", "tool_search_call":
		base.EventType = "tool_call"
		base.Tool = &session.SessionTool{
			CallID:    payload.CallID,
			Name:      firstNonEmpty(payload.Name, payload.Type),
			Namespace: payload.Namespace,
			Arguments: argumentsText(payload),
			Status:    payload.Status,
		}
		base.Content = base.Tool.Arguments
		return base
	case "function_call_output", "custom_tool_call_output", "tool_search_output":
		base.EventType = "tool_result"
		result := outputText(payload)
		base.Tool = &session.SessionTool{
			CallID: payload.CallID,
			Name:   payload.Type,
			Result: result,
			Status: firstNonEmpty(payload.Status, inferStatus(result)),
		}
		base.Content = result
		return base
	case "reasoning":
		base.EventType = "reasoning"
		base.Content = reasoningSummary(payload.Summary)
		if len(payload.Summary) > 0 {
			base.Metadata["summary"] = compactJSON(payload.Summary)
		}
		return base
	default:
		base.EventType = "unknown"
		base.Content = compactJSON(raw)
		return base
	}
}

func parseEventMsg(base session.SessionEvent, raw json.RawMessage) session.SessionEvent {
	var payload typedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		base.EventType = "unknown"
		base.Content = compactJSON(raw)
		base.Metadata["parse_warning"] = err.Error()
		return base
	}
	base.Metadata["event_msg_type"] = payload.Type
	switch payload.Type {
	case "user_message":
		base.EventType = "user_message"
		base.Role = "user"
		base.Content = payload.Message
	case "agent_message":
		base.EventType = "agent_message"
		base.Role = "assistant"
		base.Content = payload.Message
	case "mcp_tool_call_end":
		base.EventType = "tool_result"
		base.Tool = &session.SessionTool{
			CallID: payload.CallID,
			Name:   "mcp_tool_call",
			Result: compactJSON(payload.Result),
			Status: inferStatus(compactJSON(payload.Result)),
		}
		base.Content = base.Tool.Result
	default:
		base.EventType = payload.Type
		base.Content = compactJSON(raw)
	}
	return base
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, block := range blocks {
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return compactJSON(raw)
}

func argumentsText(payload typedPayload) string {
	if len(payload.Arguments) > 0 {
		var s string
		if err := json.Unmarshal(payload.Arguments, &s); err == nil {
			return s
		}
		return compactJSON(payload.Arguments)
	}
	if len(payload.Input) > 0 {
		return compactJSON(payload.Input)
	}
	return ""
}

func outputText(payload typedPayload) string {
	if len(payload.Output) > 0 {
		var s string
		if err := json.Unmarshal(payload.Output, &s); err == nil {
			return s
		}
		return compactJSON(payload.Output)
	}
	if len(payload.Tools) > 0 {
		return compactJSON(payload.Tools)
	}
	return ""
}

func reasoningSummary(raw json.RawMessage) string {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, block := range blocks {
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
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
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15-04-05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, strings.Replace(value, "Z", "+00:00", 1)); err == nil {
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

var exitCodePattern = regexp.MustCompile(`Exit code:\s*([0-9]+)`)

func inferStatus(output string) string {
	if m := exitCodePattern.FindStringSubmatch(output); len(m) == 2 && m[1] != "0" {
		return "FAILED"
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception") {
		return "FAILED"
	}
	return "ok"
}

func summarizeArguments(name, raw string) string {
	if raw == "" {
		return fmt.Sprintf("[%s]", name)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		for _, key := range []string{"description", "command", "query", "pattern", "path"} {
			if value, ok := obj[key].(string); ok && value != "" {
				return fmt.Sprintf("[%s] %s", name, session.Truncate(value, 80))
			}
		}
	}
	return fmt.Sprintf("[%s] %s", name, session.Truncate(raw, 80))
}
