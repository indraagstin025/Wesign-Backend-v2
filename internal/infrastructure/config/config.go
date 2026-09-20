// Package config menyediakan objek konfigurasi terpusat yang dibaca dari
// variabel lingkungan (+ berkas .env opsional) dan divalidasi secara
// fail-fast saat startup (TDD §2.9).
//
// Acuan nama variabel: `.env.example` (roadmap Fase 1 O-4), dengan fallback
// nama resmi TDD §2.9 (LOG_LEVEL, KMS_KEY_ID, CLAMAV_HOST, TURNSTILE_SECRET).
//
// Prinsip keamanan (roadmap Fase 1 §4.1):
//   - String() menyamarkan seluruh kredensial (URL password, S3 key, JWT).
//   - Validate() hanya dijalankan penuh di lingkungan production: JWT_SECRET
//     min 32 karakter, DATABASE_URL/REDIS_URL jelas bukan default/localhost,
//     kredensial S3 bukan `minioadmin`, JWT issuer/audience tidak kosong.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Konstanta default yang dipakai untuk deteksi production misconfig.
const (
	DefaultDatabaseURL = "postgres://localhost:5432/wesign?sslmode=disable"
	DefaultRedisURL    = "redis://localhost:6379/0"
	DefaultJWTSecret   = "default-insecure-dev-secret-change-in-production-min-32-chars"
	BaseURLDefault     = "http://localhost:8080/api/v1"
)

// Config menampung seluruh konfigurasi aplikasi WeSign.
type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	SMTP      SMTPConfig
	S3        S3Config
	KMS       KMSConfig
	CA        CAConfig
	River     RiverConfig
	ClamAV    ClamAVConfig
	Turnstile TurnstileConfig
	RateLimit RateLimitConfig
	OTEL      OTELConfig
	Argon2    Argon2Config
	TOTP      TOTPConfig
}

type AppConfig struct {
	Name     string
	Env      string // development | staging | production
	Port     int
	BaseURL  string
	LogLevel string // debug | info | warn | error
}

func (a *AppConfig) IsProduction() bool {
	return a.Env == "production"
}

func (a *AppConfig) IsDevelopment() bool {
	return a.Env == "development"
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
	Audience   string
}

type SMTPConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	FromEmail string
	FromName  string
}

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

type KMSConfig struct {
	Provider     string // local | aws
	LocalKeyPath string
	KeyID        string // TDD §2.9: KMS_KEY_ID (alias AWS_KMS_KEY_ID)
	AWSKMSKeyID  string // pencocokan backward compatibility
	AWSRegion    string
}

type OTELConfig struct {
	Endpoint string
}

type CAConfig struct {
	CertPath string
	KeyPath  string
}

type RiverConfig struct {
	PollInterval time.Duration
	MaxWorkers   int
}

type ClamAVConfig struct {
	URL     string
	Enabled bool
}

type TurnstileConfig struct {
	Enabled   bool
	SiteKey   string
	SecretKey string
}

type RateLimitConfig struct {
	Enabled                bool // Fail-Safe Default (TDD §1.5): aktif secara default
	LoginAttempts          int
	LoginWindow            time.Duration
	RegisterAttempts       int
	RegisterWindow         time.Duration
	ForgotPasswordAttempts int
	ForgotPasswordWindow   time.Duration
	DocumentLimit          int
	DocumentWindow         time.Duration
	VerifyLimit            int
	VerifyWindow           time.Duration
	SigningOTPLimit        int
	SigningOTPWindow       time.Duration
	APILimit               int
	APIWindow              time.Duration
}

type Argon2Config struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen int
}

type TOTPConfig struct {
	Issuer string
	Period uint
	Digits int
}

// Load membaca konfigurasi dari berkas .env (opsional, beberapa berkas dapat
// dilisitakan) dan variabel lingkungan sistem. Tidak panic bila berkas .env
// tidak ada — aman dijalankan di kontainer yang hanya memakai environment.
func Load(envFiles ...string) (*Config, error) {
	if len(envFiles) > 0 {
		_ = godotenv.Load(envFiles...)
	} else {
		_ = godotenv.Load()
	}

	// O-4: log level — TDD §2.9 nama LOG_LEVEL, .env.example APP_LOG_LEVEL.
	// Default dependente dari lingkungan (TDD §2.6): dev=debug, altri=info.
	logLevel := getEnv("LOG_LEVEL", "")
	if logLevel == "" {
		logLevel = getEnv("APP_LOG_LEVEL", "")
	}
	if logLevel == "" {
		env := getEnv("APP_ENV", "development")
		if env == "development" {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	// O-4: KMS key ID — TDD §2.9 KMS_KEY_ID, .env.example AWS_KMS_KEY_ID.
	kmsKeyID := getEnv("KMS_KEY_ID", "")
	if kmsKeyID == "" {
		kmsKeyID = getEnv("AWS_KMS_KEY_ID", "")
	}

	cfg := &Config{
		App: AppConfig{
			Name:     getEnv("APP_NAME", "wesign"),
			Env:      getEnv("APP_ENV", "development"),
			Port:     getEnvAsInt("APP_PORT", 8080),
			BaseURL:  getEnv("APP_BASE_URL", BaseURLDefault),
			LogLevel: logLevel,
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", DefaultDatabaseURL),
			MaxOpenConns:    getEnvAsInt("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", DefaultRedisURL),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", DefaultJWTSecret),
			AccessTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getEnvAsDuration("JWT_REFRESH_TTL", 168*time.Hour),
			Issuer:     getEnv("JWT_ISSUER", "wesign.id"),
			Audience:   getEnv("JWT_AUDIENCE", "wesign-api"),
		},
		SMTP: SMTPConfig{
			Host:      getEnv("SMTP_HOST", "localhost"),
			Port:      getEnvAsInt("SMTP_PORT", 1025),
			User:      getEnv("SMTP_USER", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", "noreply@wesign.local"),
			FromName:  getEnv("SMTP_FROM_NAME", "WeSign"),
		},
		S3: S3Config{
			Endpoint:  getEnv("S3_ENDPOINT", "http://localhost:9000"),
			AccessKey: getEnv("S3_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("S3_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("S3_BUCKET", "wesign-documents"),
			Region:    getEnv("S3_REGION", "ap-southeast-1"),
			UseSSL:    getEnvAsBool("S3_USE_SSL", false),
		},
		KMS: KMSConfig{
			Provider:     getEnv("KMS_PROVIDER", "local"),
			LocalKeyPath: getEnv("KMS_LOCAL_KEY_PATH", "./tmp/kms/dev-root.key"),
			KeyID:        kmsKeyID,
			AWSKMSKeyID:  kmsKeyID,
			AWSRegion:    getEnv("AWS_REGION", "ap-southeast-1"),
		},
		CA: CAConfig{
			CertPath: getEnv("CA_CERT_PATH", "./tmp/ca/ca.crt"),
			KeyPath:  getEnv("CA_KEY_PATH", "./tmp/ca/ca.key"),
		},
		River: RiverConfig{
			PollInterval: getEnvAsDuration("RIVER_POLL_INTERVAL", 1*time.Second),
			MaxWorkers:   getEnvAsInt("RIVER_MAX_WORKERS", 10),
		},
		ClamAV: ClamAVConfig{
			// O-4: CLAMAV_URL (.env.example) dengan fallback CLAMAV_HOST (TDD §2.9).
			URL:     getEnvDual("CLAMAV_URL", "CLAMAV_HOST", "tcp://localhost:3310"),
			Enabled: getEnvAsBool("CLAMAV_ENABLED", false),
		},
		Turnstile: TurnstileConfig{
			Enabled:   getEnvAsBool("TURNSTILE_ENABLED", false),
			SiteKey:   getEnv("TURNSTILE_SITE_KEY", ""),
			SecretKey: getEnvDual("TURNSTILE_SECRET_KEY", "TURNSTILE_SECRET", ""),
		},
		RateLimit: RateLimitConfig{
			Enabled:                getEnvAsBool("RATE_LIMIT_ENABLED", true),
			LoginAttempts:          getEnvAsInt("RATE_LIMIT_LOGIN", 5),
			LoginWindow:            getEnvAsDuration("RATE_LIMIT_LOGIN_WINDOW", 15*time.Minute),
			RegisterAttempts:       getEnvAsInt("RATE_LIMIT_REGISTER", 5),
			RegisterWindow:         getEnvAsDuration("RATE_LIMIT_REGISTER_WINDOW", 15*time.Minute),
			ForgotPasswordAttempts: getEnvAsInt("RATE_LIMIT_FORGOT_PASSWORD", 3),
			ForgotPasswordWindow:   getEnvAsDuration("RATE_LIMIT_FORGOT_PASSWORD_WINDOW", 15*time.Minute),
			DocumentLimit:          getEnvAsInt("RATE_LIMIT_DOCUMENT", 20),
			DocumentWindow:         getEnvAsDuration("RATE_LIMIT_DOCUMENT_WINDOW", 60*time.Minute),
			VerifyLimit:            getEnvAsInt("RATE_LIMIT_VERIFY", 10),
			VerifyWindow:           getEnvAsDuration("RATE_LIMIT_VERIFY_WINDOW", 60*time.Second),
			SigningOTPLimit:        getEnvAsInt("RATE_LIMIT_SIGNING_OTP", 3),
			SigningOTPWindow:       getEnvAsDuration("RATE_LIMIT_SIGNING_OTP_WINDOW", 15*time.Minute),
			APILimit:               getEnvAsInt("RATE_LIMIT_API", 120),
			APIWindow:              getEnvAsDuration("RATE_LIMIT_API_WINDOW", 1*time.Minute),
		},
		OTEL: OTELConfig{
			Endpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
		Argon2: Argon2Config{
			Memory:  getEnvAsUint32("ARGON2_MEMORY", 64*1024),
			Time:    getEnvAsUint32("ARGON2_TIME", 3),
			Threads: getEnvAsUint8("ARGON2_THREADS", 4),
			KeyLen:  getEnvAsUint32("ARGON2_KEY_LEN", 32),
			SaltLen: getEnvAsInt("ARGON2_SALT_LEN", 16),
		},
		TOTP: TOTPConfig{
			Issuer: getEnv("TOTP_ISSUER", "WeSign"),
			Period: getEnvAsUint("TOTP_PERIOD", 30),
			Digits: getEnvAsInt("TOTP_DIGITS", 6),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate memeriksa kecukupan konfigurasi penting di lingkungan production.
// Di lingkungan development/staging validasi tidak berjalankan agar developer
// dapat mengambarinan dengan nilai default (fail-fast hanya untuk production).
func (c *Config) Validate() error {
	if !c.App.IsProduction() {
		return nil
	}

	// --- JWT (roadmap Fase 1 §4.1, M-03) ---
	if c.JWT.Secret == "" || c.JWT.Secret == DefaultJWTSecret {
		return fmt.Errorf("JWT_SECRET must be securely set in production")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if c.JWT.Issuer == "" {
		return fmt.Errorf("JWT_ISSUER must not be empty in production")
	}
	if c.JWT.Audience == "" {
		return fmt.Errorf("JWT_AUDIENCE must not be empty in production")
	}

	// --- Database ---
	if c.Database.URL == "" || c.Database.URL == DefaultDatabaseURL {
		return fmt.Errorf("DATABASE_URL must be explicitly set in production (default is not allowed)")
	}
	if isLocalhostURL(c.Database.URL) {
		return fmt.Errorf("DATABASE_URL must not point to localhost in production")
	}

	// --- Redis ---
	if c.Redis.URL == "" || c.Redis.URL == DefaultRedisURL {
		return fmt.Errorf("REDIS_URL must be explicitly set in production (default is not allowed)")
	}
	if isLocalhostURL(c.Redis.URL) {
		return fmt.Errorf("REDIS_URL must not point to localhost in production")
	}

	// --- S3 (temuan audit Fase 1: tetapkan bukan minioadmin) ---
	if c.S3.AccessKey == "minioadmin" || c.S3.SecretKey == "minioadmin" {
		return fmt.Errorf("S3 credentials cannot use default minioadmin in production")
	}

	return nil
}

// isLocalhostURL memeriksa apakah URL menunjuk ke localhost/loopback.
func isLocalhostURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "localhost") ||
		strings.Contains(lower, "127.0.0.1") ||
		strings.Contains(lower, "::1")
}

// String menyediakan reprezentasi aman untuk log (NFR: kredensial di-mask).
// Tidak pernah menampahkan password URL, kredensial S3, atau JWT plaintext.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{App:{Name:%s, Env:%s, Port:%d, LogLevel:%s}, DB:%s, Redis:%s, S3:{bucket:%s, access_key:***}, JWT:{len:%d}}",
		c.App.Name,
		c.App.Env,
		c.App.Port,
		c.App.LogLevel,
		redactURL(c.Database.URL),
		redactURL(c.Redis.URL),
		c.S3.Bucket,
		len(c.JWT.Secret),
	)
}

// redactURL menyamarkan kredensial di URL menjadi [REDACTED] bila parsing gagal,
// atau nilai .Redacted() bila URL dapat diparse.
func redactURL(raw string) string {
	if raw == "" {
		return "[EMPTY]"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "[REDACTED]"
	}
	return parsed.Redacted()
}

// getEnv membaca variabel lingkungan; bila tidak ada, mengembalikan defaultVal.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// getEnvDual membaca variabel primer (acuan .env.example); bila kosong,
// fallback ke variabel sekunder (nama resmi TDD §2.9) — solusi butir O-4.
func getEnvDual(primary, secondary, defaultVal string) string {
	if val := getEnv(primary, ""); val != "" {
		return val
	}
	return getEnv(secondary, defaultVal)
}

func getEnvAsInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvAsUint32(key string, defaultVal uint32) uint32 {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if uVal, err := strconv.ParseUint(val, 10, 32); err == nil {
			return uint32(uVal)
		}
	}
	return defaultVal
}

func getEnvAsUint8(key string, defaultVal uint8) uint8 {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if uVal, err := strconv.ParseUint(val, 10, 8); err == nil {
			return uint8(uVal)
		}
	}
	return defaultVal
}

func getEnvAsUint(key string, defaultVal uint) uint {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if uVal, err := strconv.ParseUint(val, 10, 32); err == nil {
			return uint(uVal)
		}
	}
	return defaultVal
}