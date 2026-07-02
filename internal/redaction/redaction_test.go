package redaction

import (
	"strings"
	"testing"
)

func TestRedactSecrets_GivenConfigSecrets_ThenMasksValues(t *testing.T) {
	input := `{"api_key":"sk-test-redaction-token-1234567890","password":"letmein","client_secret":"abc1234567890"}`
	got := RedactSecrets(input)
	for _, leaked := range []string{"sk-test-redaction-token-1234567890", "letmein", "abc1234567890"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("secret %q leaked in %q", leaked, got)
		}
	}
	if !strings.Contains(got, "[REDACTED_SECRET]") && !strings.Contains(got, "[REDACTED_TOKEN]") {
		t.Fatalf("redacted placeholders missing in %q", got)
	}
}

func TestRedactSecrets_GivenBearerJWTAndPEM_ThenMasksThem(t *testing.T) {
	input := strings.Join([]string{
		"Authorization: Bearer abcdefghijklmnopqrstuvwxyz",
		"jwt=eyJaaaaaaaaaaa.bbbbbbbbbbbb.cccccccccccc",
		"-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----",
	}, "\n")
	got := RedactSecrets(input)
	for _, want := range []string{"[REDACTED_TOKEN]", "[REDACTED_JWT]", "[REDACTED_PRIVATE_KEY]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("redacted output missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "abcdefghijklmnopqrstuvwxyz") {
		t.Fatalf("sensitive content leaked in %q", got)
	}
}
