// Package analyzer provides stats and audit analysis over parsed transcripts.
package analyzer

import (
	"strings"
	"unicode/utf8"

	"claude-code-session-reader/internal/jsonutil"
	"claude-code-session-reader/internal/parser"
	"claude-code-session-reader/internal/summarizer"
)

type StatsResult struct {
	RawText       string
	FilteredText  string
	RawChars      int
	FilteredChars int
	Categories    map[string]int
}

func ComputeStats(entries []map[string]interface{}) StatsResult {
	var rawParts, filteredParts []string
	categories := map[string]int{
		"user_text":       0,
		"user_answers":    0,
		"assistant_text":  0,
		"tool_summaries":  0,
		"tool_input_raw":  0,
		"tool_result_raw": 0,
		"system_noise":    0,
	}

	for _, entry := range entries {
		message, ok := entry["message"].(map[string]interface{})
		if !ok {
			continue
		}

		if parser.IsNoise(entry) {
			text := parser.ExtractAllText(entry)
			categories["system_noise"] += utf8.RuneCountInString(text)
			rawParts = append(rawParts, text)
			continue
		}

		content := message["content"]

		if _, hasToolResult := entry["toolUseResult"]; hasToolResult {
			full := parser.ExtractAllText(entry)
			if summarizer.IsUserAnswer(entry) {
				answer := summarizer.ExtractUserAnswers(entry)
				categories["user_answers"] += utf8.RuneCountInString(answer)
				rawParts = append(rawParts, full)
				filteredParts = append(filteredParts, answer)
			} else {
				categories["tool_result_raw"] += utf8.RuneCountInString(full)
				rawParts = append(rawParts, full)
				summary := summarizer.SummarizeToolResult(entry)
				categories["tool_summaries"] += utf8.RuneCountInString(summary)
				filteredParts = append(filteredParts, summary)
			}
			continue
		}

		role := jsonutil.GetStr(message, "role")
		switch role {
		case "user":
			text := parser.ExtractText(content)
			if strings.TrimSpace(text) != "" {
				categories["user_text"] += utf8.RuneCountInString(text)
				rawParts = append(rawParts, text)
				filteredParts = append(filteredParts, text)
			}
		case "assistant":
			text := parser.ExtractText(content)
			if strings.TrimSpace(text) != "" {
				categories["assistant_text"] += utf8.RuneCountInString(text)
				rawParts = append(rawParts, text)
				filteredParts = append(filteredParts, text)
			}
			for _, tb := range parser.GetToolUses(content) {
				rawJSON := jsonutil.MarshalNoEscape(jsonutil.GetInputMap(tb))
				categories["tool_input_raw"] += utf8.RuneCountInString(rawJSON)
				rawParts = append(rawParts, rawJSON)

				name := jsonutil.GetStr(tb, "name")
				if name == "" {
					name = "?"
				}
				summary := summarizer.SummarizeToolUse(name, jsonutil.GetInputMap(tb))
				categories["tool_summaries"] += utf8.RuneCountInString(summary)
				filteredParts = append(filteredParts, summary)
			}
		}
	}

	rawText := strings.Join(rawParts, "\n")
	filteredText := strings.Join(filteredParts, "\n")

	return StatsResult{
		RawText:       rawText,
		FilteredText:  filteredText,
		RawChars:      utf8.RuneCountInString(rawText),
		FilteredChars: utf8.RuneCountInString(filteredText),
		Categories:    categories,
	}
}
