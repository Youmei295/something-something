package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully-resolved runtime configuration for the API and worker.
// It is populated from environment variables so that the same binary can run in
// development, CI, and production without code changes.
type Config struct {
	Env             string
	HTTP            HTTPConfig
	Gateway         GatewayConfig
	DatabaseURL     string
	Storage         StorageConfig
	Mailer          MailerConfig
	Auth            AuthConfig
	Host            HostConfig
	Limits          LimitsConfig
	Worker          WorkerConfig
	ShutdownTimeout time.Duration
}

type HTTPConfig struct {
	Addr           string
	PublicBaseURL  string
	AllowedOrigins []string
}

// GatewayConfig configures the API gateway that fronts the individual services.
type GatewayConfig struct {
	Addr            string
	IdentityURL     string
	SubmissionURL   string
	ConversationURL string
	AttachmentURL   string
}

type StorageConfig struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	UseSSL         bool
	PresignTTL     time.Duration
}

type MailerConfig struct {
	Driver          string // console | smtp | postmark
	FromAddress     string
	FromName        string
	PostmarkToken   string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPUseSTARTTLS bool
}

type AuthConfig struct {
	SessionTTL   time.Duration
	CookieName   string
	CookieSecure bool
	CookieDomain string
	VisitorTTL   time.Duration
}

type HostConfig struct {
	Email string
	Name  string
	// Password is only consumed by the seed command; it is never hashed at load time.
	Password string
}

type LimitsConfig struct {
	MaxUploadBytes      int64
	AllowedContentTypes []string
}

type WorkerConfig struct {
	PollInterval time.Duration
	Concurrency  int
}

// Load reads configuration from the environment and applies safe defaults.
//
// Before reading, it optionally loads a dotenv file so `go run` works locally
// without exporting variables by hand. Real environment variables always win.
// The file is chosen from ENV_FILE, falling back to ".env" in the working
// directory.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		Env:         env("APP_ENV", "development"),
		DatabaseURL: env("DATABASE_URL", "postgres://inbox:inbox@localhost:5432/inbox?sslmode=disable"),
		HTTP: HTTPConfig{
			Addr:           env("HTTP_ADDR", ":8080"),
			PublicBaseURL:  strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:3000"), "/"),
			AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		Gateway: GatewayConfig{
			Addr:            env("GATEWAY_ADDR", ":8080"),
			IdentityURL:     env("GATEWAY_IDENTITY_URL", "http://localhost:8081"),
			SubmissionURL:   env("GATEWAY_SUBMISSION_URL", "http://localhost:8082"),
			ConversationURL: env("GATEWAY_CONVERSATION_URL", "http://localhost:8083"),
			AttachmentURL:   env("GATEWAY_ATTACHMENT_URL", "http://localhost:8084"),
		},
		Storage: StorageConfig{
			Endpoint:       env("STORAGE_ENDPOINT", "localhost:9000"),
			PublicEndpoint: env("STORAGE_PUBLIC_ENDPOINT", ""),
			AccessKey:      env("STORAGE_ACCESS_KEY", "minioadmin"),
			SecretKey:      env("STORAGE_SECRET_KEY", "minioadmin"),
			Bucket:         env("STORAGE_BUCKET", "attachments"),
			Region:         env("STORAGE_REGION", "us-east-1"),
			UseSSL:         envBool("STORAGE_USE_SSL", false),
			PresignTTL:     envDuration("STORAGE_PRESIGN_TTL", 15*time.Minute),
		},
		Mailer: MailerConfig{
			Driver:          env("MAILER_DRIVER", "console"),
			FromAddress:     env("MAIL_FROM_ADDRESS", "inbox@example.com"),
			FromName:        env("MAIL_FROM_NAME", "Inbox"),
			PostmarkToken:   env("POSTMARK_SERVER_TOKEN", ""),
			SMTPHost:        env("SMTP_HOST", ""),
			SMTPPort:        envInt("SMTP_PORT", 587),
			SMTPUsername:    env("SMTP_USERNAME", ""),
			SMTPPassword:    env("SMTP_PASSWORD", ""),
			SMTPUseSTARTTLS: envBool("SMTP_USE_STARTTLS", true),
		},
		Auth: AuthConfig{
			SessionTTL:   envDuration("SESSION_TTL", 7*24*time.Hour),
			CookieName:   env("SESSION_COOKIE_NAME", "inbox_session"),
			CookieSecure: envBool("SESSION_COOKIE_SECURE", false),
			CookieDomain: env("SESSION_COOKIE_DOMAIN", ""),
			VisitorTTL:   envDuration("VISITOR_TOKEN_TTL", 90*24*time.Hour),
		},
		Host: HostConfig{
			Email:    env("HOST_EMAIL", "host@example.com"),
			Name:     env("HOST_NAME", "Host"),
			Password: env("HOST_PASSWORD", ""),
		},
		Limits: LimitsConfig{
			MaxUploadBytes:      envInt64("MAX_UPLOAD_BYTES", 5<<20),
			AllowedContentTypes: splitCSV(env("ALLOWED_CONTENT_TYPES", "image/png,image/jpeg,image/gif,image/webp,application/pdf")),
		},
		Worker: WorkerConfig{
			PollInterval: envDuration("WORKER_POLL_INTERVAL", 2*time.Second),
			Concurrency:  envInt("WORKER_CONCURRENCY", 4),
		},
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 20*time.Second),
	}

	if cfg.Mailer.Driver == "postmark" && cfg.Mailer.PostmarkToken == "" {
		return nil, fmt.Errorf("MAILER_DRIVER=postmark requires POSTMARK_SERVER_TOKEN")
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

// loadDotEnv applies KEY=VALUE pairs from a dotenv file to the process
// environment, only when the key is not already set. It is best-effort: a
// missing file is not an error, so production (env-only) is unaffected.
func loadDotEnv() {
	path := os.Getenv("ENV_FILE")
	if path == "" {
		path = ".env"
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envInt64(key string, fallback int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
