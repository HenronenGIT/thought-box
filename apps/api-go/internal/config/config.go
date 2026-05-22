package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	CorsAllowedOrigins []string
	LogLevel           slog.Level
	Limits             Limits
	S3                 S3
	OpenAIAPIKey       string
	WorkerEnabled      bool
}

type Limits struct {
	MaxDurationMs int64 `json:"max_duration_ms"`
	MinDurationMs int64 `json:"min_duration_ms"`
	MaxSizeBytes  int64 `json:"max_size_bytes"`
}

type S3 struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func FromEnv() (Config, error) {
	return fromMap(os.Environ())
}

func fromMap(environ []string) (Config, error) {
	env := map[string]string{}
	for _, pair := range environ {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			env[key] = value
		}
	}

	databaseURL, err := required(env, "DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	databaseURL, err = normalizeDatabaseURL(databaseURL)
	if err != nil {
		return Config{}, err
	}
	if err := validateDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	openAIAPIKey, err := required(env, "OPENAI_API_KEY")
	if err != nil {
		return Config{}, err
	}
	s3Bucket, err := required(env, "S3_BUCKET")
	if err != nil {
		return Config{}, err
	}
	s3Region, err := required(env, "S3_REGION")
	if err != nil {
		return Config{}, err
	}
	accessKeyID, err := required(env, "AWS_ACCESS_KEY_ID")
	if err != nil {
		return Config{}, err
	}
	secretAccessKey, err := required(env, "AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:             optional(env, "APP_ENV", "dev"),
		Port:               optional(env, "PORT", "8080"),
		DatabaseURL:        databaseURL,
		CorsAllowedOrigins: splitCSV(optional(env, "CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		LogLevel:           parseLogLevel(optional(env, "LOG_LEVEL", "info")),
		Limits: Limits{
			MaxDurationMs: optionalInt64(env, "MAX_THOUGHT_DURATION_MS", 60_000),
			MinDurationMs: optionalInt64(env, "MIN_THOUGHT_DURATION_MS", 1_000),
			MaxSizeBytes:  optionalInt64(env, "MAX_THOUGHT_SIZE_BYTES", 10*1024*1024),
		},
		S3: S3{
			Bucket:          s3Bucket,
			Region:          s3Region,
			Endpoint:        strings.TrimSpace(env["S3_ENDPOINT"]),
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
		},
		OpenAIAPIKey:  openAIAPIKey,
		WorkerEnabled: optionalBool(env, "WORKER_ENABLED", true),
	}, nil
}

func (c Config) DatabaseHost() string {
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "unknown"
	}
	return u.Hostname()
}

func required(env map[string]string, name string) (string, error) {
	value := strings.TrimSpace(env[name])
	if value == "" {
		return "", fmt.Errorf("missing required env var: %s", name)
	}
	return value, nil
}

func optional(env map[string]string, name string, fallback string) string {
	value := strings.TrimSpace(env[name])
	if value == "" {
		return fallback
	}
	return value
}

func optionalInt64(env map[string]string, name string, fallback int64) int64 {
	value := strings.TrimSpace(env[name])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func optionalBool(env map[string]string, name string, fallback bool) bool {
	value := strings.TrimSpace(env[name])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func validateDatabaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use postgres or postgresql scheme")
	}
	if u.User == nil || u.User.Username() == "" {
		return fmt.Errorf("DATABASE_URL must include username")
	}
	if _, ok := u.User.Password(); !ok {
		return fmt.Errorf("DATABASE_URL must include password")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("DATABASE_URL must include host")
	}
	return nil
}

func normalizeDatabaseURL(value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	query := u.Query()
	if query.Get("default_query_exec_mode") == "" {
		query.Set("default_query_exec_mode", "simple_protocol")
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
