package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16

	MaxPasswordLength = 1024
	MinMemoryLimit    = 8 * 1024
	MaxMemoryLimit    = 256 * 1024
	MinTimeLimit      = 1
	MaxTimeLimit      = 10
	MinThreadsLimit   = 1
	MaxThreadsLimit   = 16
	MinHashKeyLength  = 16
	MaxHashKeyLength  = 64
	MinSaltLength     = 8
	MaxSaltLength     = 64
)

// argonParams menampung hasil parsing PHC string.
type argonParams struct {
	version int
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	hash    []byte
}

// Hash menghasilkan Argon2id hash dalam format PHC string menggunakan parameter default rekomendasi WeSign.
func Hash(password string) (string, error) {
	return HashWithCustomParams(password, argonMemory, argonTime, argonThreads, argonKeyLen, saltLen)
}

// HashWithCustomParams menghasilkan Argon2id hash dalam format PHC string dengan parameter kustom
// yang divalidasi terhadap batas keamanan DoS (Min/Max limits).
func HashWithCustomParams(password string, memory, time uint32, threads uint8, keyLen uint32, sLen int) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	if len(password) > MaxPasswordLength {
		return "", fmt.Errorf("password exceeds maximum allowed length of %d bytes", MaxPasswordLength)
	}
	if memory < MinMemoryLimit || memory > MaxMemoryLimit {
		return "", fmt.Errorf("memory parameter %d out of safe bounds [%d, %d]", memory, MinMemoryLimit, MaxMemoryLimit)
	}
	if time < MinTimeLimit || time > MaxTimeLimit {
		return "", fmt.Errorf("time parameter %d out of safe bounds [%d, %d]", time, MinTimeLimit, MaxTimeLimit)
	}
	if threads < MinThreadsLimit || threads > MaxThreadsLimit {
		return "", fmt.Errorf("threads parameter %d out of safe bounds [%d, %d]", threads, MinThreadsLimit, MaxThreadsLimit)
	}
	if keyLen < MinHashKeyLength || keyLen > MaxHashKeyLength {
		return "", fmt.Errorf("key length parameter %d out of safe bounds [%d, %d]", keyLen, MinHashKeyLength, MaxHashKeyLength)
	}
	if sLen < MinSaltLength || sLen > MaxSaltLength {
		return "", fmt.Errorf("salt length parameter %d out of safe bounds [%d, %d]", sLen, MinSaltLength, MaxSaltLength)
	}

	salt := make([]byte, sLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads, b64Salt, b64Hash,
	), nil
}

// Verify memverifikasi kesesuaian password plaintext terhadap Argon2id PHC encoded hash.
func Verify(password, encodedHash string) (bool, error) {
	if err := validateVerifyInputs(password, encodedHash); err != nil {
		return false, err
	}

	params, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Panjang hash sudah divalidasi di [MinHashKeyLength, MaxHashKeyLength] = [16, 64],
	// sehingga konversi ke uint32 aman dari overflow.
	keyLen := uint32(len(params.hash)) //nolint:gosec // G115: bounded by decodeSaltAndHash
	computed := argon2.IDKey(
		[]byte(password),
		params.salt,
		params.time,
		params.memory,
		params.threads,
		keyLen,
	)

	return subtle.ConstantTimeCompare(params.hash, computed) == 1, nil
}

func validateVerifyInputs(password, encodedHash string) error {
	if password == "" || encodedHash == "" {
		return fmt.Errorf("password and encoded hash cannot be empty")
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password exceeds maximum allowed length of %d bytes", MaxPasswordLength)
	}
	return nil
}

func parseArgon2idHash(encodedHash string) (*argonParams, error) {
	parts, err := splitPHC(encodedHash)
	if err != nil {
		return nil, err
	}

	p := &argonParams{}
	if err := parsePHCParams(parts, p); err != nil {
		return nil, err
	}
	if err := validateParamsBounds(p); err != nil {
		return nil, err
	}
	if err := decodeSaltAndHash(parts, p); err != nil {
		return nil, err
	}

	return p, nil
}

// splitPHC memvalidasi struktur dasar dan mengembalikan 6 segmen.
func splitPHC(encodedHash string) ([]string, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid hash format: expected 6 segments, got %d", len(parts))
	}
	if parts[0] != "" {
		return nil, fmt.Errorf("invalid hash format: prefix must be empty, got %q", parts[0])
	}
	if parts[1] != "argon2id" {
		return nil, fmt.Errorf("invalid hash algorithm: expected argon2id, got %q", parts[1])
	}
	return parts, nil
}

// parsePHCParams membaca versi dan parameter dari segmen PHC.
func parsePHCParams(parts []string, p *argonParams) error {
	if _, err := fmt.Sscanf(parts[2], "v=%d", &p.version); err != nil {
		return fmt.Errorf("parse version: %w", err)
	}
	if p.version != argon2.Version {
		return fmt.Errorf("unsupported argon2 version: got %d, expected %d", p.version, argon2.Version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return fmt.Errorf("parse params: %w", err)
	}
	return nil
}

// validateParamsBounds memastikan seluruh parameter berada di rentang aman.
func validateParamsBounds(p *argonParams) error {
	if p.memory < MinMemoryLimit || p.memory > MaxMemoryLimit {
		return fmt.Errorf("memory parameter %d out of safe bounds [%d, %d]",
			p.memory, MinMemoryLimit, MaxMemoryLimit)
	}
	if p.time < MinTimeLimit || p.time > MaxTimeLimit {
		return fmt.Errorf("time parameter %d out of safe bounds [%d, %d]",
			p.time, MinTimeLimit, MaxTimeLimit)
	}
	if p.threads < MinThreadsLimit || p.threads > MaxThreadsLimit {
		return fmt.Errorf("threads parameter %d out of safe bounds [%d, %d]",
			p.threads, MinThreadsLimit, MaxThreadsLimit)
	}
	return nil
}

// decodeSaltAndHash men-decode salt dan hash dari base64 dan memvalidasi panjangnya.
func decodeSaltAndHash(parts []string, p *argonParams) error {
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}
	if len(salt) < MinSaltLength || len(salt) > MaxSaltLength {
		return fmt.Errorf("salt length %d out of safe bounds [%d, %d]",
			len(salt), MinSaltLength, MaxSaltLength)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("decode hash: %w", err)
	}
	if len(hash) < MinHashKeyLength || len(hash) > MaxHashKeyLength {
		return fmt.Errorf("hash key length %d out of safe bounds [%d, %d]",
			len(hash), MinHashKeyLength, MaxHashKeyLength)
	}

	p.salt = salt
	p.hash = hash
	return nil
}
