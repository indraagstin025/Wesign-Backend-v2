// Package masking menyediakan fungsi penyamaran (masking) data sensitif
// untuk keperluan logging dan tampilan di UI.
//
// Rujukan: TDD §8.11, SRS FR-CERT-06, NFR-25; Roadmap Fase 1 §4.5.
package masking

import (
	"regexp"
	"strings"
)

// octetPattern cocok untuk nilai 0–255 (tanpa leading zero berlebih).
const octetPattern = `(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)`

var ipv4Regex = regexp.MustCompile(
	`^(` + octetPattern + `\.` + octetPattern + `\.` + octetPattern + `)\.` + octetPattern + `$`,
)

// MaskEmail menyamarkan email: user@example.com → u***@example.com
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "***"
	}
	local := parts[0]
	domain := parts[1]

	if len(local) <= 1 {
		return "*@" + domain
	}
	return string(local[0]) + "***@" + domain
}

// MaskNIK menyamarkan NIK 16 digit: 3201234567890001 → **** **** **** 0001
func MaskNIK(nik string) string {
	cleanNIK := strings.ReplaceAll(nik, " ", "")
	if len(cleanNIK) != 16 {
		return "****"
	}
	return "**** **** **** " + cleanNIK[12:]
}

const maskedIPv6 = "xxxx:xxxx"

// MaskIP menyamarkan segmen terakhir alamat IPv4/IPv6.
// IPv4: 192.168.1.100 → 192.168.1.xxx
// IPv6: 2001:db8::1 → prefix:xxxx:xxxx
// Invalid: → xxx.xxx.xxx.xxx
func MaskIP(ip string) string {
	trimmed := strings.TrimSpace(ip)

	if ipv4Regex.MatchString(trimmed) {
		return ipv4Regex.ReplaceAllString(trimmed, "$1.xxx")
	}

	if strings.Contains(trimmed, ":") {
		if trimmed == "::1" || trimmed == "::" {
			return maskedIPv6
		}
		parts := strings.Split(trimmed, ":")
		if len(parts) > 2 {
			prefix := strings.Trim(strings.Join(parts[:len(parts)-2], ":"), ":")
			if prefix == "" {
				return maskedIPv6
			}
			return prefix + ":" + maskedIPv6
		}
		return maskedIPv6
	}

	return "xxx.xxx.xxx.xxx"
}

// MaskPhone menyamarkan nomor telepon: +628123456789 → +62812****789
func MaskPhone(phone string) string {
	clean := strings.TrimSpace(phone)
	if len(clean) < 8 {
		return "***"
	}
	prefixLen := 7
	if len(clean) < 10 {
		prefixLen = 4
	}
	return clean[:prefixLen] + "****" + clean[len(clean)-3:]
}

// MaskSecret menyamarkan token/API key untuk logging aman.
func MaskSecret(secret string) string {
	clean := strings.TrimSpace(secret)
	if len(clean) <= 8 {
		return "********"
	}
	return clean[:4] + "********" + clean[len(clean)-4:]
}
