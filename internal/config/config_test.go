package config

import (
	"strings"
	"testing"
)

// setRequired sets the minimum required env var so other tests can focus
// on a single field at a time without DATABASE_URL interfering.
func withDatabaseURL(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/alerting")
}

// --- Load() ---

func TestLoad_AllRequiredVarsSet(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/alerting")
	t.Setenv("NATS_URL", "nats://nats:4222")
	t.Setenv("REDIS_URL", "redis://redis:6379")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if cfg.DatabaseUrl != "postgres://user:pass@localhost:5432/alerting" {
		t.Errorf("unexpected DatabaseUrl: %q", cfg.DatabaseUrl)
	}
	if cfg.NATSUrl != "nats://nats:4222" {
		t.Errorf("unexpected NATSUrl: %q", cfg.NATSUrl)
	}
	if cfg.RedisUrl != "redis://redis:6379" {
		t.Errorf("unexpected RedisUrl: %q", cfg.RedisUrl)
	}
}

func TestLoad_DatabaseURL_Missing(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is unset")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %q", err.Error())
	}
}

func TestLoad_DatabaseURL_WhitespaceOnly(t *testing.T) {
	// Whitespace-only value must be treated as unset.
	t.Setenv("DATABASE_URL", "   ")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for whitespace-only DATABASE_URL")
	}
}

func TestLoad_Defaults_Applied(t *testing.T) {
	withDatabaseURL(t)
	// Unset all optional vars to confirm defaults are applied.
	for _, key := range []string{
		"NATS_URL", "REDIS_URL", "API_ADDR", "LOG_LEVEL", "SMTP_PORT", "WSS_URL",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NATSUrl != "nats://localhost:4222" {
		t.Errorf("expected default NATS_URL, got: %q", cfg.NATSUrl)
	}
	if cfg.RedisUrl != "redis://localhost:6379" {
		t.Errorf("expected default REDIS_URL, got: %q", cfg.RedisUrl)
	}
	if cfg.APIAddr != ":8081" {
		t.Errorf("expected default API_ADDR, got: %q", cfg.APIAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LOG_LEVEL, got: %q", cfg.LogLevel)
	}
	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTP_PORT 587, got: %d", cfg.SMTPPort)
	}
	if cfg.WSSUrl != "ws://localhost:8080/ws" {
		t.Errorf("expected default WSS_URL %q, got: %q", "ws://localhost:8080/ws", cfg.WSSUrl)
	}
}

func TestLoad_SMTPPort_CustomValue(t *testing.T) {
	withDatabaseURL(t)
	t.Setenv("SMTP_PORT", "465")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SMTPPort != 465 {
		t.Errorf("expected SMTP_PORT 465, got: %d", cfg.SMTPPort)
	}
}

func TestLoad_SMTPPort_InvalidValue(t *testing.T) {
	withDatabaseURL(t)
	t.Setenv("SMTP_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-integer SMTP_PORT")
	}
}

func TestLoad_OptionalFields_Populated(t *testing.T) {
	withDatabaseURL(t)
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_USER", "user@example.com")
	t.Setenv("SMTP_PASS", "secret")
	t.Setenv("TWILIO_SID", "ACxxx")
	t.Setenv("TWILIO_TOKEN", "token123")
	t.Setenv("TWILIO_FROM", "+15550000000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("unexpected SMTPHost: %q", cfg.SMTPHost)
	}
	if cfg.SMTPPass != "secret" {
		t.Errorf("unexpected SMTPPass: %q", cfg.SMTPPass)
	}
	if cfg.TwilioToken != "token123" {
		t.Errorf("unexpected TwilioToken: %q", cfg.TwilioToken)
	}
}

func TestLoad_SMTPPort_Whitespace(t *testing.T) {
	// Whitespace is non-empty so it bypasses the default and hits strconv.Atoi,
	// which fails — Load() must return an error.
	withDatabaseURL(t)
	t.Setenv("SMTP_PORT", "   ")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for whitespace-only SMTP_PORT")
	}
}

func TestLoad_SMTPPort_Zero(t *testing.T) {
	// "0" is a valid integer; the implementation stores it without range validation.
	// This test documents the current behaviour: zero port is accepted.
	withDatabaseURL(t)
	t.Setenv("SMTP_PORT", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error for SMTP_PORT=0: %v", err)
	}
	if cfg.SMTPPort != 0 {
		t.Errorf("expected SMTPPort 0, got: %d", cfg.SMTPPort)
	}
}

func TestLoad_OptionalURL_WhitespaceValue_NotReplacedByDefault(t *testing.T) {
	// envOr treats only the empty string as "use default".
	// A whitespace-only value is considered explicitly set and stored as-is.
	withDatabaseURL(t)
	t.Setenv("NATS_URL", "   ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NATSUrl != "   " {
		t.Errorf("expected whitespace NATS_URL to be preserved as-is, got: %q", cfg.NATSUrl)
	}
}

// --- String() masking ---

func TestString_MasksSMTPPass(t *testing.T) {
	cfg := &Config{
		DatabaseUrl: "postgres://localhost/db",
		SMTPPass:    "super-secret-password",
	}
	out := cfg.String()
	if strings.Contains(out, "super-secret-password") {
		t.Errorf("String() must not contain the raw SMTPPass value, got: %s", out)
	}
	if !strings.Contains(out, "***") {
		t.Errorf("String() should contain *** as mask for non-empty SMTPPass, got: %s", out)
	}
}

func TestString_MasksTwilioToken(t *testing.T) {
	cfg := &Config{
		DatabaseUrl: "postgres://localhost/db",
		TwilioToken: "twilio-auth-token-xyz",
	}
	out := cfg.String()
	if strings.Contains(out, "twilio-auth-token-xyz") {
		t.Errorf("String() must not contain the raw TwilioToken value, got: %s", out)
	}
	if !strings.Contains(out, "***") {
		t.Errorf("String() should contain *** as mask for non-empty TwilioToken, got: %s", out)
	}
}

func TestString_EmptySecrets_NotMasked(t *testing.T) {
	// When secrets are empty, String() must not emit *** (there is nothing to hide).
	cfg := &Config{
		DatabaseUrl: "postgres://localhost/db",
		SMTPPass:    "",
		TwilioToken: "",
	}
	out := cfg.String()
	if strings.Contains(out, "***") {
		t.Errorf("String() should not mask empty secrets, got: %s", out)
	}
}

func TestString_BothSecretsPresent(t *testing.T) {
	cfg := &Config{
		DatabaseUrl: "postgres://localhost/db",
		SMTPPass:    "pass1",
		TwilioToken: "tok1",
	}
	out := cfg.String()
	if strings.Contains(out, "pass1") {
		t.Errorf("SMTPPass must be masked in String()")
	}
	if strings.Contains(out, "tok1") {
		t.Errorf("TwilioToken must be masked in String()")
	}
}

// --- WSSUrl ---

func TestLoad_WSSUrl_Custom(t *testing.T) {
	withDatabaseURL(t)
	t.Setenv("WSS_URL", "wss://feed.example.com/stream")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WSSUrl != "wss://feed.example.com/stream" {
		t.Errorf("WSSUrl: want %q, got %q", "wss://feed.example.com/stream", cfg.WSSUrl)
	}
}

func TestLoad_WSSUrl_WhitespacePreserved(t *testing.T) {
	// envOr only replaces the empty string with the default; a non-empty whitespace
	// value is an explicit (if unusual) override and must be stored as-is.
	withDatabaseURL(t)
	t.Setenv("WSS_URL", "   ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WSSUrl != "   " {
		t.Errorf("whitespace WSS_URL must be preserved as-is, got: %q", cfg.WSSUrl)
	}
}

func TestString_ContainsWSSUrl(t *testing.T) {
	// WSSUrl is not a secret; it must appear verbatim and unmasked in String().
	cfg := &Config{
		DatabaseUrl: "postgres://localhost/db",
		WSSUrl:      "wss://feed.example.com/stream",
	}
	out := cfg.String()
	if !strings.Contains(out, "wss://feed.example.com/stream") {
		t.Errorf("String() must contain WSSUrl unmasked, got: %s", out)
	}
}
