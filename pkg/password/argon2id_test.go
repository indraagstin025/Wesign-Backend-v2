package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// TEST ASLI
// ============================================================================

func TestArgon2id_HashAndVerify(t *testing.T) {
	rawPassword := "SuperSecretPassword123!"

	hash, err := Hash(rawPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "$argon2id$v=19$m=65536,t=3,p=4$")

	match, err := Verify(rawPassword, hash)
	require.NoError(t, err)
	assert.True(t, match)

	matchWrong, err := Verify("WrongPassword123!", hash)
	require.NoError(t, err)
	assert.False(t, matchWrong)
}

func TestArgon2id_DifferentSaltEachTime(t *testing.T) {
	rawPassword := "SamePasswordAcrossRuns"

	hash1, err := Hash(rawPassword)
	require.NoError(t, err)
	hash2, err := Hash(rawPassword)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Salts must be randomly generated so hashes differ")

	match1, err := Verify(rawPassword, hash1)
	require.NoError(t, err)
	assert.True(t, match1)

	match2, err := Verify(rawPassword, hash2)
	require.NoError(t, err)
	assert.True(t, match2)
}

func TestArgon2id_InvalidInputs(t *testing.T) {
	t.Run("empty password on hash", func(t *testing.T) {
		_, err := Hash("")
		assert.Error(t, err)
	})

	t.Run("empty inputs on verify", func(t *testing.T) {
		_, err := Verify("", "hash")
		assert.Error(t, err)

		_, err = Verify("pass", "")
		assert.Error(t, err)
	})

	t.Run("malformed hash on verify", func(t *testing.T) {
		invalidHashes := []string{
			"invalid-format",
			"$argon2$v=19$m=65536,t=3,p=4$salt$hash",
			"$argon2id$v=invalid$m=65536,t=3,p=4$salt$hash",
			"$argon2id$v=19$invalid_params$salt$hash",
			"$argon2id$v=19$m=65536,t=3,p=4$invalid!base64$hash",
			"$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$invalid!hash",
		}

		for _, h := range invalidHashes {
			match, err := Verify("password", h)
			assert.Error(t, err, "hash %q harus error", h)
			assert.False(t, match)
		}
	})

	t.Run("password exceeds max allowed length", func(t *testing.T) {
		hugePassword := string(make([]byte, 1025))
		_, err := Hash(hugePassword)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed length")

		match, err := Verify(hugePassword, "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.False(t, match)
	})

	t.Run("malicious parameters out of safe bounds on verify", func(t *testing.T) {
		_, err := Verify("password", "$argon2id$v=19$m=1024,t=3,p=4$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory parameter")

		_, err = Verify("password", "$argon2id$v=19$m=524288,t=3,p=4$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory parameter")

		_, err = Verify("password", "$argon2id$v=19$m=65536,t=0,p=4$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time parameter")

		_, err = Verify("password", "$argon2id$v=19$m=65536,t=20,p=4$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time parameter")

		_, err = Verify("password", "$argon2id$v=19$m=65536,t=3,p=0$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "threads parameter")

		_, err = Verify("password", "$argon2id$v=19$m=65536,t=3,p=32$c2FsdA$aGFzaA")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "threads parameter")
	})
}

// ============================================================================
// REGRESSION TEST — Perilaku setelah fix
// ============================================================================

const regressionSalt = "MTIzNDU2Nzg5MDEyMzQ1Ng" // "1234567890123456" base64 raw

// Version != argon2.Version HARUS ditolak dengan error versi.
func TestRegression_Argon2id_RejectsUnsupportedVersion(t *testing.T) {
	legacyHash := "$argon2id$v=18$m=65536,t=3,p=4$" + regressionSalt + "$aGFzaA"

	match, err := Verify("anypassword", legacyHash)
	require.Error(t, err, "v=18 harus error")
	assert.False(t, match)
	assert.Contains(t, err.Error(), "unsupported argon2 version")
}

// parts[0] tidak boleh non-empty.
func TestRegression_Argon2id_RejectsNonEmptyPrefix(t *testing.T) {
	badPrefixHash := "garbage$argon2id$v=19$m=65536,t=3,p=4$" + regressionSalt + "$aGFzaA"

	match, err := Verify("password", badPrefixHash)
	require.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "prefix must be empty")
}

// Panjang key hash harus divalidasi.
func TestRegression_Argon2id_RejectsShortHashKey(t *testing.T) {
	tinyKeyHash := "$argon2id$v=19$m=65536,t=3,p=4$" + regressionSalt + "$YQ"

	match, err := Verify("password", tinyKeyHash)
	require.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "hash key length")
}

// Panjang key hash maksimum juga divalidasi.
func TestRegression_Argon2id_RejectsLongHashKey(t *testing.T) {
	// key 100 byte base64 raw = 134 karakter
	longKey := strings.Repeat("QUJD", 50) // 200 chars → ~150 bytes
	// Sesuaikan agar tepat > 64 byte setelah decode
	longKeyHash := "$argon2id$v=19$m=65536,t=3,p=4$" + regressionSalt + "$" + longKey

	match, err := Verify("password", longKeyHash)
	require.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "hash key length")
}

// Panjang salt harus divalidasi (minimal 8 byte).
func TestRegression_Argon2id_RejectsShortSalt(t *testing.T) {
	// Salt "abc" (3 byte) di base64 raw = YWJj
	shortSaltHash := "$argon2id$v=19$m=65536,t=3,p=4$YWJj$aGFzaA"

	match, err := Verify("password", shortSaltHash)
	require.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "salt length")
}

// Algoritma selain argon2id harus ditolak.
func TestRegression_Argon2id_RejectsWrongAlgorithm(t *testing.T) {
	wrongAlgoHash := "$argon2i$v=19$m=65536,t=3,p=4$" + regressionSalt + "$aGFzaA"

	match, err := Verify("password", wrongAlgoHash)
	require.Error(t, err)
	assert.False(t, match)
	assert.Contains(t, err.Error(), "invalid hash algorithm")
}

// Format hash yang valid harus tetap sukses (happy path).
func TestRegression_Argon2id_ValidHashStillWorks(t *testing.T) {
	raw := "ValidPassword123!"
	hash, err := Hash(raw)
	require.NoError(t, err)

	match, err := Verify(raw, hash)
	require.NoError(t, err)
	assert.True(t, match)

	matchWrong, err := Verify("wrong", hash)
	require.NoError(t, err)
	assert.False(t, matchWrong)
}

func TestArgon2id_HashWithCustomParams(t *testing.T) {
	raw := "CustomParamPassword123!"

	t.Run("happy path with custom valid parameters", func(t *testing.T) {
		// Use minimum valid memory and time for faster unit test execution
		hash, err := HashWithCustomParams(raw, MinMemoryLimit, MinTimeLimit, MinThreadsLimit, MinHashKeyLength, MinSaltLength)
		require.NoError(t, err)
		assert.Contains(t, hash, "$argon2id$v=19$m=8192,t=1,p=1$")

		match, err := Verify(raw, hash)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("empty password rejected", func(t *testing.T) {
		_, err := HashWithCustomParams("", 64*1024, 3, 4, 32, 16)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})

	t.Run("password exceeds max length rejected", func(t *testing.T) {
		huge := string(make([]byte, MaxPasswordLength+1))
		_, err := HashWithCustomParams(huge, 64*1024, 3, 4, 32, 16)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed length")
	})

	t.Run("memory bounds rejection", func(t *testing.T) {
		_, errLow := HashWithCustomParams(raw, MinMemoryLimit-1, 3, 4, 32, 16)
		assert.Error(t, errLow)
		assert.Contains(t, errLow.Error(), "memory parameter")

		_, errHigh := HashWithCustomParams(raw, MaxMemoryLimit+1, 3, 4, 32, 16)
		assert.Error(t, errHigh)
		assert.Contains(t, errHigh.Error(), "memory parameter")
	})

	t.Run("time bounds rejection", func(t *testing.T) {
		_, errLow := HashWithCustomParams(raw, 64*1024, MinTimeLimit-1, 4, 32, 16)
		assert.Error(t, errLow)
		assert.Contains(t, errLow.Error(), "time parameter")

		_, errHigh := HashWithCustomParams(raw, 64*1024, MaxTimeLimit+1, 4, 32, 16)
		assert.Error(t, errHigh)
		assert.Contains(t, errHigh.Error(), "time parameter")
	})

	t.Run("threads bounds rejection", func(t *testing.T) {
		_, errLow := HashWithCustomParams(raw, 64*1024, 3, 0, 32, 16)
		assert.Error(t, errLow)
		assert.Contains(t, errLow.Error(), "threads parameter")

		_, errHigh := HashWithCustomParams(raw, 64*1024, 3, MaxThreadsLimit+1, 32, 16)
		assert.Error(t, errHigh)
		assert.Contains(t, errHigh.Error(), "threads parameter")
	})

	t.Run("key length bounds rejection", func(t *testing.T) {
		_, errLow := HashWithCustomParams(raw, 64*1024, 3, 4, MinHashKeyLength-1, 16)
		assert.Error(t, errLow)
		assert.Contains(t, errLow.Error(), "key length parameter")

		_, errHigh := HashWithCustomParams(raw, 64*1024, 3, 4, MaxHashKeyLength+1, 16)
		assert.Error(t, errHigh)
		assert.Contains(t, errHigh.Error(), "key length parameter")
	})

	t.Run("salt length bounds rejection", func(t *testing.T) {
		_, errLow := HashWithCustomParams(raw, 64*1024, 3, 4, 32, MinSaltLength-1)
		assert.Error(t, errLow)
		assert.Contains(t, errLow.Error(), "salt length parameter")

		_, errHigh := HashWithCustomParams(raw, 64*1024, 3, 4, 32, MaxSaltLength+1)
		assert.Error(t, errHigh)
		assert.Contains(t, errHigh.Error(), "salt length parameter")
	})
}
