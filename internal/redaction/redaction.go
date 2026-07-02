// Package redaction masks common secret shapes before filtered content leaves
// the local process boundary.
package redaction

import "regexp"

var patterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{
		re:   regexp.MustCompile(`(?is)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
		repl: "[REDACTED_PRIVATE_KEY]",
	},
	{
		re:   regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{12,}`),
		repl: "Bearer [REDACTED_TOKEN]",
	},
	{
		re:   regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
		repl: "[REDACTED_JWT]",
	},
	{
		re:   regexp.MustCompile(`\b(?:sk|sk-ant|ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_=-]{12,}\b`),
		repl: "[REDACTED_TOKEN]",
	},
	{
		re:   regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{12,}\b`),
		repl: "[REDACTED_TOKEN]",
	},
	{
		re:   regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		repl: "[REDACTED_AWS_KEY]",
	},
	{
		re:   regexp.MustCompile(`(?i)\b[a-z0-9_-]*(?:secret|token|key)[a-z0-9_-]*[=:][A-Za-z0-9+/=_-]{24,}\b`),
		repl: "[REDACTED_HIGH_ENTROPY_SECRET]",
	},
	{
		re:   regexp.MustCompile(`(?i)("?(?:api[_-]?key|access[_-]?token|refresh[_-]?token|token|password|passwd|pwd|cookie|client[_-]?secret|authorization)"?\s*[:=]\s*)("[^"]+"|'[^']+'|[^\s,}\]]+)`),
		repl: "${1}[REDACTED_SECRET]",
	},
}

func RedactSecrets(text string) string {
	for _, pattern := range patterns {
		text = pattern.re.ReplaceAllString(text, pattern.repl)
	}
	return text
}
