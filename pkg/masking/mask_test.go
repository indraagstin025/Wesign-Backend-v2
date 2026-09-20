package masking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "u***@example.com"},
		{"john.doe@wesign.id", "j***@wesign.id"},
		{"a@example.com", "*@example.com"},
		{"invalid-email", "***"},
		{"", "***"},
		{"@domain.com", "***"},
		{"user@", "***"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskEmail(tt.input))
		})
	}
}

func TestMaskNIK(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"3201234567890001", "**** **** **** 0001"},
		{"3201 2345 6789 0001", "**** **** **** 0001"},
		{"12345", "****"},
		{"12345678901234567", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskNIK(tt.input))
		})
	}
}

func TestMaskIP_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard", "192.168.1.100", "192.168.1.xxx"},
		{"private", "10.0.0.1", "10.0.0.xxx"},
		{"localhost", "127.0.0.1", "127.0.0.xxx"},
		{"zeros", "0.0.0.0", "0.0.0.xxx"},
		{"broadcast", "255.255.255.255", "255.255.255.xxx"},
		{"invalid octet", "999.999.999.999", "xxx.xxx.xxx.xxx"},
		{"not ip", "invalid-ip", "xxx.xxx.xxx.xxx"},
		{"empty", "", "xxx.xxx.xxx.xxx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskIP(tt.input))
		})
	}
}

func TestMaskIP_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:0000:8a2e:xxxx:xxxx"},
		{"loopback", "::1", "xxxx:xxxx"},
		{"unspecified", "::", "xxxx:xxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskIP(tt.input))
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"international", "+628123456789", "+628123****789"},
		{"local long", "081234567890", "0812345****890"},
		{"short 8 digits", "12345678", "1234****678"},
		{"too short", "12345", "***"},
		{"empty", "", "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskPhone(tt.input))
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"long token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "eyJh********VCJ9"},
		{"exactly 9 chars", "123456789", "1234********6789"},
		{"8 chars", "12345678", "********"},
		{"short", "abc", "********"},
		{"empty", "", "********"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskSecret(tt.input))
		})
	}
}
