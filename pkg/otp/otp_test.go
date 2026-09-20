package otp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// TEST ASLI
// ============================================================================

func TestOTP_Generate(t *testing.T) {
	numericRegex := regexp.MustCompile(`^\d{6}$`)

	for i := 0; i < 50; i++ {
		code, hash, err := Generate()
		require.NoError(t, err)
		assert.Len(t, code, 6)
		assert.True(t, numericRegex.MatchString(code), "OTP must be exactly 6 numeric digits")
		assert.Len(t, hash, 64, "SHA-256 hex must be 64 chars")
		assert.Equal(t, HashOTP(code), hash)
	}
}

func TestOTP_ValidateConstantTime(t *testing.T) {
	code, hash, err := Generate()
	require.NoError(t, err)

	assert.True(t, ValidateConstantTime(code, hash))

	wrongCode := "999999"
	if code == "999999" {
		wrongCode = "000000"
	}
	assert.False(t, ValidateConstantTime(wrongCode, hash))

	assert.False(t, ValidateConstantTime("123", hash))
	assert.False(t, ValidateConstantTime("1234567", hash))
	assert.False(t, ValidateConstantTime("", hash))
	assert.False(t, ValidateConstantTime(code, ""))
}

func TestOTP_HMACWithPepper(t *testing.T) {
	pepper := "super-secret-server-side-pepper-key"
	code, hmacHash, err := Generate(pepper)
	require.NoError(t, err)

	assert.True(t, ValidateConstantTime(code, hmacHash, pepper))

	plainHash := HashOTP(code)
	assert.NotEqual(t, plainHash, hmacHash)

	assert.False(t, ValidateConstantTime(code, hmacHash, "wrong-pepper-key"))

	assert.False(t, ValidateConstantTime(code, hmacHash))
}

// ============================================================================
// REGRESSION TEST — Perilaku setelah fix
// ============================================================================

// Empty pepper eksplisit HARUS error, bukan fallback diam-diam ke SHA-256.
func TestRegression_OTP_EmptyPepperReturnsError(t *testing.T) {
	_, _, err := Generate("")
	require.Error(t, err, "Generate(\"\") harus error, bukan silent fallback")
	assert.Contains(t, err.Error(), "pepper must not be empty string")
}

// ValidateConstantTime dengan empty pepper HARUS false.
func TestRegression_OTP_ValidateEmptyPepperReturnsFalse(t *testing.T) {
	code, hash, err := Generate()
	require.NoError(t, err)

	valid := ValidateConstantTime(code, hash, "")
	assert.False(t, valid, "ValidateConstantTime dengan empty pepper harus false")
}

// Lebih dari 1 pepper HARUS error.
func TestRegression_OTP_RejectsMultiplePeppers(t *testing.T) {
	_, _, err := Generate("pepper1", "pepper2")
	require.Error(t, err, "Generate dengan >1 pepper harus error")
	assert.Contains(t, err.Error(), "at most one pepper")
}

// Validasi dengan >1 pepper HARUS false.
func TestRegression_OTP_ValidateMultiplePeppersReturnsFalse(t *testing.T) {
	pepper := "valid-pepper"
	code, hash, err := Generate(pepper)
	require.NoError(t, err)

	valid := ValidateConstantTime(code, hash, pepper, "extra-pepper")
	assert.False(t, valid, "ValidateConstantTime dengan >1 pepper harus false")
}

// Pepper valid tetap bekerja seperti sebelumnya.
func TestRegression_OTP_ValidPepperStillWorks(t *testing.T) {
	pepper := "correct-server-side-pepper"

	code, hash, err := Generate(pepper)
	require.NoError(t, err)
	require.Len(t, code, 6)
	require.Len(t, hash, 64)

	assert.True(t, ValidateConstantTime(code, hash, pepper))
	assert.False(t, ValidateConstantTime(code, hash, "wrong-pepper"))

	// HMAC berbeda dari plain SHA-256
	plainHash := HashOTP(code)
	assert.NotEqual(t, plainHash, hash)

	// Manual HMAC-SHA256 implementation for cross-verification
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(code))
	manualHash := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, manualHash, hash, "HMAC-SHA256 manual harus cocok dengan HashOTP")
}

// ============================================================================
// INFO TEST — perilaku by-design
// ============================================================================

// Tanpa pepper, hash = SHA-256 biasa. Ini fakta desain (bukan bug),
// mitigasi utamanya adalah mewajibkan pepper di production melalui config.
func TestProof_OTP_WithoutPepper_IsPlainSHA256(t *testing.T) {
	code, storedHash, err := Generate()
	require.NoError(t, err)
	require.Len(t, code, 6)

	h := sha256.Sum256([]byte(code))
	attackerComputed := hex.EncodeToString(h[:])

	assert.Equal(t, attackerComputed, storedHash,
		"Tanpa pepper, hash OTP = SHA-256 biasa (by design, tapi tidak disarankan di production)")

	t.Log("=== INFO ===")
	t.Logf("OTP plaintext    : %s", code)
	t.Logf("Stored hash      : %s", storedHash)
	t.Logf("Attacker SHA-256 : %s", attackerComputed)
	t.Log("Rekomendasi: gunakan pepper di production.")
}
