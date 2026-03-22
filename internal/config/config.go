package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the alerting system services.
// Values are loaded from environment variables via Load().
type Config struct {
	NATSUrl     string
	RedisUrl    string
	DatabaseUrl string
	APIAddr     string
	LogLevel    string

	WSSUrl string // WebSocket feed URL for the ingestion service adapter

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string // masked in String()

	TwilioSID   string
	TwilioToken string // masked in String()
	TwilioFrom  string
}

// Load reads configuration from environment variables and returns a Config.
// DATABASE_URL is required; all other fields have defaults or are optional.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dbURL) == "" {
		return nil, errors.New("required environment variable DATABASE_URL is not set")
	}

	smtpPort := 587
	if raw := os.Getenv("SMTP_PORT"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid SMTP_PORT %q: must be an integer", raw)
		}
		smtpPort = v
	}

	cfg := &Config{
		NATSUrl:     envOr("NATS_URL", "nats://localhost:4222"),
		RedisUrl:    envOr("REDIS_URL", "redis://localhost:6379"),
		DatabaseUrl: dbURL,
		APIAddr:     envOr("API_ADDR", ":8081"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		WSSUrl:      envOr("WSS_URL", "ws://localhost:8080/ws"),

		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: smtpPort,
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),

		TwilioSID:   os.Getenv("TWILIO_SID"),
		TwilioToken: os.Getenv("TWILIO_TOKEN"),
		TwilioFrom:  os.Getenv("TWILIO_FROM"),
	}
	return cfg, nil
}

// String returns a log-safe representation of the config.
// SMTPPass and TwilioToken are masked to prevent accidental secret exposure.
func (c *Config) String() string {
	smtpPass := maskSecret(c.SMTPPass)
	twilioToken := maskSecret(c.TwilioToken)

	return fmt.Sprintf(
		"Config{NATSUrl:%q RedisUrl:%q DatabaseUrl:%q APIAddr:%q LogLevel:%q WSSUrl:%q "+
			"SMTPHost:%q SMTPPort:%d SMTPUser:%q SMTPPass:%s "+
			"TwilioSID:%q TwilioToken:%s TwilioFrom:%q}",
		c.NATSUrl, c.RedisUrl, c.DatabaseUrl, c.APIAddr, c.LogLevel, c.WSSUrl,
		c.SMTPHost, c.SMTPPort, c.SMTPUser, smtpPass,
		c.TwilioSID, twilioToken, c.TwilioFrom,
	)
}

// envOr returns the value of the named environment variable, or fallback if unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// maskSecret returns "***" when the secret is non-empty, or `""` when it is empty.
func maskSecret(s string) string {
	if s != "" {
		return "***"
	}
	return `""`
}
