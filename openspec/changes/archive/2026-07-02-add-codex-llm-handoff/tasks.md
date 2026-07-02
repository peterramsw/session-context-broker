## 1. Provider Adapters

- [x] 1.1 Define normalized `SessionEvent`/`SessionMetadata`/`SessionRef` types and stable event ID derivation
- [x] 1.2 Implement Codex session discovery/inspect/parse against real local Codex session shape
- [x] 1.3 Add Codex tests for real-or-synthetic fixture parse, malformed event, unknown event, and interrupted session
- [x] 1.4 Verify Google Antigravity standalone app session root and transcript format, distinct from Antigravity IDE storage
- [x] 1.5 Implement Antigravity standalone app discovery/inspect/parse against `~/.gemini/antigravity/brain/<conversation-id>/.system_generated/logs/{transcript_full.jsonl,transcript.jsonl}`
- [x] 1.6 Add Antigravity tests for fixture parsing, analyzer integration, malformed-line continuation, env-root discovery, and CLI filtered handoff

## 2. Provider-Aware CLI

- [x] 2.1 Add provider-aware `list --provider {claude_code|codex|antigravity|all}`
- [x] 2.2 Add provider-aware `inspect`
- [x] 2.3 Add provider-aware `filter`
- [x] 2.4 Add provider-aware `stats --provider codex|antigravity --no-tokens`
- [x] 2.5 Add provider-aware `handoff --provider`
- [x] 2.6 Re-run upstream/new tests to confirm existing commands remain unaffected

## 3. Filtered-First Handoff

- [x] 3.1 Implement `handoff --llm auto|always|never`
- [x] 3.2 Always write redacted `filtered.md` before Local LLM decision
- [x] 3.3 Use redacted filtered character count and configurable threshold for `--llm auto`
- [x] 3.4 Allow filtered-only handoff when Local LLM is absent
- [x] 3.5 Error clearly for `--llm always` when Local LLM is not configured, after writing filtered output

## 4. Optional Local LLM Handoff

- [x] 4.1 Implement configurable OpenAI-compatible client with empty API key support
- [x] 4.2 Send deterministic sampling config (`temperature`, optional `top_p`/`top_k`) and max output token settings
- [x] 4.3 Implement handoff schema structs and Markdown renderer with derived-artifact disclosure
- [x] 4.4 Implement one repair attempt for malformed JSON and fail loud if repair fails
- [x] 4.5 Verify existing token utilities before adding tokenizer dependencies

## 5. Smoke Tests and Evidence

- [x] 5.1 Record original upstream-style baseline measurements
- [x] 5.2 Record provider smoke tests for Claude Code and Codex
- [x] 5.3 Record live Local LLM smoke tests
- [x] 5.4 Record filtered-first policy smoke tests
- [x] 5.5 Record Google Antigravity standalone app smoke tests
- [x] 5.6 Run `go test ./...`, `go build ./...`, and `openspec validate add-codex-llm-handoff --strict`
