package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
)

const (
	OTPMaxRange = 1000000 // 000000 s/d 999999
	OTPLength   = 6
)

// Generate menghasilkan 6-digit OTP secara kriptografis aman.
//
// Jika pepper diberikan (string non-kosong), hash disimpan menggunakan HMAC-SHA256
// sesuai rekomendasi OWASP untuk keyspace kecil 10^6.
//
// Jika pepper tidak diberikan (variadic kosong), hash memakai SHA-256 biasa.
// Mode ini TIDAK DISARANKAN untuk production.
//
// Pepper string kosong eksplisit ("") mengembalikan error untuk menangkap
// miskonfigurasi (mis. env var yang lupa diisi).
func Generate(pepper ...string) (string, string, error) {
	if len(pepper) > 1 {
		return "", "", fmt.Errorf("otp: at most one pepper is supported, got %d", len(pepper))
	}
	if len(pepper) == 1 && pepper[0] == "" {
		return "", "", fmt.Errorf("otp: pepper must not be empty string; omit the argument for insecure default or provide a real pepper")
	}

	maxLimit := big.NewInt(OTPMaxRange)
	n, err := rand.Int(rand.Reader, maxLimit)
	if err != nil {
		return "", "", fmt.Errorf("generate random int: %w", err)
	}

	otp := fmt.Sprintf("%06d", n.Int64())
	otpHash := HashOTP(otp, pepper...)

	return otp, otpHash, nil
}

// HashOTP menghasilkan hash hexdigest dari string OTP.
//
// Jika pepper[0] non-kosong, menggunakan HMAC-SHA256.
// Jika tidak, menggunakan SHA-256 biasa.
//
// Catatan: hanya pepper[0] yang dipakai. Elemen pepper tambahan diabaikan
// demi kompatibilitas backward API variadic.
func HashOTP(otp string, pepper ...string) string {
	if len(pepper) > 0 && pepper[0] != "" {
		mac := hmac.New(sha256.New, []byte(pepper[0]))
		mac.Write([]byte(otp))
		return hex.EncodeToString(mac.Sum(nil))
	}
	h := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(h[:])
}

// ValidateConstantTime memverifikasi OTP masukan terhadap hash tersimpan secara timing-safe.
//
// Pepper string kosong eksplisit mengembalikan false (bukan fallback ke SHA-256),
// agar tidak ada mismatch diam-diam dengan hash yang dibuat memakai pepper.
func ValidateConstantTime(inputOTP, storedHash string, pepper ...string) bool {
	if len(inputOTP) != OTPLength || storedHash == "" {
		return false
	}
	if len(pepper) > 1 {
		return false
	}
	if len(pepper) == 1 && pepper[0] == "" {
		return false
	}
	inputHash := HashOTP(inputOTP, pepper...)
	return subtle.ConstantTimeCompare([]byte(inputHash), []byte(storedHash)) == 1
}
