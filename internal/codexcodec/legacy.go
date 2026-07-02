package codexcodec

import (
	"encoding/json"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func legacyEvents(events []session.SessionEvent) []session.Event {
	var out []session.Event
	for _, event := range events {
		switch event.EventType {
		case "message":
			switch event.Role {
			case "user":
				out = append(out, session.Event{
					Kind:      session.EventUserMessage,
					Timestamp: event.Timestamp,
					RawType:   "codex_message",
					User:      &session.UserMessage{Text: event.Content},
				})
			case "assistant":
				out = append(out, session.Event{
					Kind:      session.EventAssistantMessage,
					Timestamp: event.Timestamp,
					RawType:   "codex_message",
					Assistant: &session.AssistantMessage{Text: event.Content},
				})
			default:
				out = append(out, noiseEvent(event))
			}
		case "tool_call":
			if event.Tool == nil {
				out = append(out, noiseEvent(event))
				continue
			}
			out = append(out, session.Event{
				Kind:      session.EventAssistantMessage,
				Timestamp: event.Timestamp,
				RawType:   "codex_tool_call",
				Assistant: &session.AssistantMessage{ToolUses: []session.ToolUse{{
					ID:    event.Tool.CallID,
					Name:  event.Tool.Name,
					Input: session.ToolInput{Raw: parseToolInput(event.Tool.Arguments)},
				}}},
			})
		case "tool_result":
			if event.Tool == nil {
				out = append(out, noiseEvent(event))
				continue
			}
			out = append(out, session.Event{
				Kind:      session.EventToolResult,
				Timestamp: event.Timestamp,
				RawType:   "codex_tool_result",
				Tool: &session.ToolResult{
					ToolUseID: event.Tool.CallID,
					Success:   event.Tool.Status != "FAILED",
					Text:      event.Tool.Result,
					RawName:   event.Tool.Name,
				},
			})
		case "reasoning", "session_meta", "turn_context":
			out = append(out, noiseEvent(event))
		default:
			if strings.TrimSpace(event.Content) != "" {
				out = append(out, noiseEvent(event))
			}
		}
	}
	return out
}

func parseToolInput(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj
	}
	return map[string]any{"arguments": raw}
}

func noiseEvent(event session.SessionEvent) session.Event {
	text := event.Content
	if text == "" && len(event.Metadata) > 0 {
		text = session.MarshalNoEscape(event.Metadata)
	}
	return session.Event{
		Kind:      session.EventNoise,
		Timestamp: event.Timestamp,
		RawType:   event.EventType,
		Noise:     &session.NoiseEvent{Text: text},
	}
}
