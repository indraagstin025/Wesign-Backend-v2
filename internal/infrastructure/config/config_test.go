package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv menghapus semua variabel lingkungan yang dibaca oleh Load()
// agar test tidak terpengaruh environment lokal.
func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"APP_NAME", "APP_ENV", "APP_PORT", "APP_BASE_URL",
		"APP_LOG_LEVEL", "LOG_LEVEL",
		"DATABASE_URL", "DATABASE_MAX_OPEN_CONNS", "DATABASE_MAX_IDLE_CONNS", "DATABASE_CONN_MAX_LIFETIME",
		"REDIS_URL", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_ACCESS_TTL", "JWT_REFRESH_TTL", "JWT_ISSUER", "JWT_AUDIENCE",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD", "SMTP_FROM_EMAIL", "SMTP_FROM_NAME",
		"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET", "S3_REGION", "S3_USE_SSL",
		"KMS_PROVIDER", "KMS_LOCAL_KEY_PATH", "KMS_KEY_ID", "AWS_KMS_KEY_ID", "AWS_REGION",
		"CA_CERT_PATH", "CA_KEY_PATH",
		"RIVER_POLL_INTERVAL", "RIVER_MAX_WORKERS",
		"CLAMAV_URL", "CLAMAV_HOST", "CLAMAV_ENABLED",
		"TURNSTILE_ENABLED", "TURNSTILE_SITE_KEY", "TURNSTILE_SECRET_KEY", "TURNSTILE_SECRET",
		"RATE_LIMIT_ENABLED", "RATE_LIMIT_LOGIN", "RATE_LIMIT_LOGIN_WINDOW",
		"RATE_LIMIT_REGISTER", "RATE_LIMIT_REGISTER_WINDOW",
		"RATE_LIMIT_FORGOT_PASSWORD", "RATE_LIMIT_FORGOT_PASSWORD_WINDOW",
		"RATE_LIMIT_DOCUMENT", "RATE_LIMIT_DOCUMENT_WINDOW",
		"RATE_LIMIT_VERIFY", "RATE_LIMIT_VERIFY_WINDOW",
		"RATE_LIMIT_SIGNING_OTP", "RATE_LIMIT_SIGNING_OTP_WINDOW",
		"RATE_LIMIT_API", "RATE_LIMIT_API_WINDOW",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"ARGON2_MEMORY", "ARGON2_TIME", "ARGON2_THREADS", "ARGON2_KEY_LEN", "ARGON2_SALT_LEN",
		"TOTP_ISSUER", "TOTP_PERIOD", "TOTP_DIGITS",
	}
	for _, k := range envVars {
		os.Unsetenv(k)
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "wesign", cfg.App.Name)
	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, 8080, cfg.App.Port)
	assert.Equal(t, "debug", cfg.App.LogLevel)
	assert.Equal(t, BaseURLDefault, cfg.App.BaseURL)
	assert.True(t, cfg.App.IsDevelopment())
	assert.False(t, cfg.App.IsProduction())
	assert.Equal(t, DefaultDatabaseURL, cfg.Database.URL)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 10, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, DefaultRedisURL, cfg.Redis.URL)
	assert.Equal(t, 15*time.Minute, cfg.JWT.AccessTTL)
	assert.Equal(t, 168*time.Hour, cfg.JWT.RefreshTTL)
	assert.Equal(t, "wesign.id", cfg.JWT.Issuer)
	assert.Equal(t, "wesign-api", cfg.JWT.Audience)
	assert.Equal(t, 1025, cfg.SMTP.Port)
	assert.Equal(t, "wesign-documents", cfg.S3.Bucket)
	assert.False(t, cfg.S3.UseSSL)
	assert.Equal(t, "", cfg.KMS.KeyID)
	assert.Equal(t, "", cfg.OTEL.Endpoint)
	assert.False(t, cfg.ClamAV.Enabled)
	assert.False(t, cfg.Turnstile.Enabled)
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 5, cfg.RateLimit.LoginAttempts)
	assert.Equal(t, 15*time.Minute, cfg.RateLimit.LoginWindow)
	assert.Equal(t, 5, cfg.RateLimit.RegisterAttempts)
	assert.Equal(t, 3, cfg.RateLimit.ForgotPasswordAttempts)
	assert.Equal(t, 20, cfg.RateLimit.DocumentLimit)
	assert.Equal(t, 60*time.Minute, cfg.RateLimit.DocumentWindow)
	assert.Equal(t, 10, cfg.RateLimit.VerifyLimit)
	assert.Equal(t, 60*time.Second, cfg.RateLimit.VerifyWindow)
	assert.Equal(t, 3, cfg.RateLimit.SigningOTPLimit)
	assert.Equal(t, 120, cfg.RateLimit.APILimit)
	assert.Equal(t, uint32(64*1024), cfg.Argon2.Memory)
	assert.Equal(t, uint32(3), cfg.Argon2.Time)
	assert.Equal(t, uint8(4), cfg.Argon2.Threads)
	assert.Equal(t, uint32(32), cfg.Argon2.KeyLen)
	assert.Equal(t, 16, cfg.Argon2.SaltLen)
	assert.Equal(t, "WeSign", cfg.TOTP.Issuer)
	assert.Equal(t, uint(30), cfg.TOTP.Period)
	assert.Equal(t, 6, cfg.TOTP.Digits)
}

func TestConfig_LogLevelPerEnvironment(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"development", "debug"},
		{"staging", "info"},
		{"production", "info"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("APP_ENV", tt.env)
			// Production & staging require valid credentials to pass Validate()
			if tt.env != "development" {
				setProductionEnv(t)
				t.Setenv("APP_ENV", tt.env) // override back after setProductionEnv
			}
			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.App.LogLevel)
		})
	}
}

func TestConfig_CustomEnvValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_NAME", "wesign-custom")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("KMS_KEY_ID", "arn:aws:kms:ap-southeast-1:123456789:key/test-key")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317")
	t.Setenv("JWT_ISSUER", "custom.wesign.id")
	t.Setenv("JWT_AUDIENCE", "custom-api")
	t.Setenv("RATE_LIMIT_REGISTER", "10")
	t.Setenv("ARGON2_MEMORY", "131072")
	t.Setenv("TOTP_ISSUER", "CustomTOTP")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("RIVER_MAX_WORKERS", "20")
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "10m")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "wesign-custom", cfg.App.Name)
	assert.Equal(t, 9090, cfg.App.Port)
	assert.Equal(t, "warn", cfg.App.LogLevel)
	assert.Equal(t, "arn:aws:kms:ap-southeast-1:123456789:key/test-key", cfg.KMS.KeyID)
	assert.Equal(t, "http://otel-collector:4317", cfg.OTEL.Endpoint)
	assert.Equal(t, "custom.wesign.id", cfg.JWT.Issuer)
	assert.Equal(t, "custom-api", cfg.JWT.Audience)
	assert.Equal(t, 10, cfg.RateLimit.RegisterAttempts)
	assert.Equal(t, uint32(131072), cfg.Argon2.Memory)
	assert.Equal(t, "CustomTOTP", cfg.TOTP.Issuer)
	assert.True(t, cfg.S3.UseSSL)
	assert.Equal(t, 20, cfg.River.MaxWorkers)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
}

func TestConfig_FallbackEnvVars_O4(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("KMS_KEY_ID", "fallback-kms-key")
	t.Setenv("CLAMAV_HOST", "http://clamav-fallback:3310")
	t.Setenv("TURNSTILE_SECRET", "fallback-turnstile-secret")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "error", cfg.App.LogLevel)
	assert.Equal(t, "fallback-kms-key", cfg.KMS.KeyID)
	assert.Equal(t, "http://clamav-fallback:3310", cfg.ClamAV.URL)
	assert.Equal(t, "fallback-turnstile-secret", cfg.Turnstile.SecretKey)
}

func TestConfig_PrimaryEnvOverridesFallback(t *testing.T) {
	clearEnv(t)
	// LOG_LEVEL = primer, APP_LOG_LEVEL = fallback (sesuai config.go baris 185-187)
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("APP_LOG_LEVEL", "error")
	// KMS_KEY_ID = primer, AWS_KMS_KEY_ID = fallback
	t.Setenv("KMS_KEY_ID", "primary-kms")
	t.Setenv("AWS_KMS_KEY_ID", "fallback-kms")
	// CLAMAV_URL = primer, CLAMAV_HOST = fallback
	t.Setenv("CLAMAV_URL", "http://primary:3310")
	t.Setenv("CLAMAV_HOST", "http://fallback:3310")
	// TURNSTILE_SECRET_KEY = primer, TURNSTILE_SECRET = fallback
	t.Setenv("TURNSTILE_SECRET_KEY", "primary-ts")
	t.Setenv("TURNSTILE_SECRET", "fallback-ts")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "warn", cfg.App.LogLevel)
	assert.Equal(t, "primary-kms", cfg.KMS.KeyID)
	assert.Equal(t, "http://primary:3310", cfg.ClamAV.URL)
	assert.Equal(t, "primary-ts", cfg.Turnstile.SecretKey)
}

func TestConfig_ProductionValidation(t *testing.T) {
	t.Run("rejects short JWT secret", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("JWT_SECRET", "short")
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
	})

	t.Run("rejects default JWT secret", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("JWT_SECRET", DefaultJWTSecret)
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET must be securely set in production")
	})

	t.Run("rejects default database URL", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("DATABASE_URL", DefaultDatabaseURL)
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_URL must be explicitly set in production")
	})

	t.Run("rejects localhost database URL", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/wesign")
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_URL must not point to localhost")
	})

	t.Run("rejects default redis URL", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("REDIS_URL", DefaultRedisURL)
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_URL must be explicitly set in production")
	})

	t.Run("rejects minioadmin S3 credentials", func(t *testing.T) {
		setProductionEnv(t)
		t.Setenv("S3_ACCESS_KEY", "minioadmin")
		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minioadmin")
	})

	t.Run("passes with valid production config", func(t *testing.T) {
		setProductionEnv(t)
		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}

func TestConfig_StringRedactsCredentials(t *testing.T) {
	clearEnv(t)
	dbPass := buildTestCredential("db")
	redisPass := buildTestCredential("redis")
	jwtSecret := buildTestCredential("jwt")

	t.Setenv("DATABASE_URL", "postgres://postgres:"+dbPass+"@prod-db:5432/wesign")
	t.Setenv("REDIS_URL", "redis://:"+redisPass+"@prod-redis:6379/0")
	t.Setenv("JWT_SECRET", jwtSecret)
	t.Setenv("S3_ACCESS_KEY", "prod-key")
	t.Setenv("S3_SECRET_KEY", "prod-secret")
	t.Setenv("JWT_AUDIENCE", "wesign-api")

	cfg, err := Load()
	require.NoError(t, err)

	str := cfg.String()
	assert.NotContains(t, str, dbPass, "DB password must be redacted")
	assert.NotContains(t, str, redisPass, "Redis password must be redacted")
	assert.NotContains(t, str, jwtSecret, "JWT secret must not appear")
	assert.Contains(t, str, "xxxxx", "redacted URL should contain xxxxx")
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getEnvAsInt invalid returns default", func(t *testing.T) {
		t.Setenv("TEST_INT", "not-a-number")
		assert.Equal(t, 42, getEnvAsInt("TEST_INT", 42))
	})

	t.Run("getEnvAsBool invalid returns default", func(t *testing.T) {
		t.Setenv("TEST_BOOL", "not-a-bool")
		assert.False(t, getEnvAsBool("TEST_BOOL", false))
	})

	t.Run("getEnvAsDuration invalid returns default", func(t *testing.T) {
		t.Setenv("TEST_DUR", "not-a-duration")
		assert.Equal(t, 5*time.Second, getEnvAsDuration("TEST_DUR", 5*time.Second))
	})

	t.Run("getEnvAsUint32 invalid returns default", func(t *testing.T) {
		t.Setenv("TEST_U32", "abc")
		assert.Equal(t, uint32(100), getEnvAsUint32("TEST_U32", 100))
	})

	t.Run("getEnvAsUint8 invalid returns default", func(t *testing.T) {
		t.Setenv("TEST_U8", "xyz")
		assert.Equal(t, uint8(4), getEnvAsUint8("TEST_U8", 4))
	})

	t.Run("isLocalhostURL detects variants", func(t *testing.T) {
		assert.True(t, isLocalhostURL("http://localhost:5432/db"))
		assert.True(t, isLocalhostURL("postgres://user:pass@127.0.0.1:5432/db"))
		assert.True(t, isLocalhostURL("redis://[::1]:6379/0"))
		assert.False(t, isLocalhostURL("postgres://prod-db:5432/wesign"))
	})
}

func setProductionEnv(t *testing.T) {
	t.Helper()
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "super-secure-production-jwt-secret-key-32-chars-minimum")
	t.Setenv("JWT_ISSUER", "wesign.id")
	t.Setenv("JWT_AUDIENCE", "wesign-api")
	t.Setenv("S3_ACCESS_KEY", "prod-access-key-12345")
	t.Setenv("S3_SECRET_KEY", "prod-secret-key-67890")
	t.Setenv("DATABASE_URL", "postgres://user:pass@prod-db:5432/wesign")
	t.Setenv("REDIS_URL", "redis://:pass@prod-redis:6379/0")
}

func buildTestCredential(kind string) string {
	return strings.Join([]string{"dummy", kind, "value", "for", "test"}, "-")
}
