// Package response menyediakan amplop JSON standar untuk API WeSign.
// Paket ini framework-agnostic (tidak mengimpor fiber/http handler).
// Handler di layer atas yang bertanggung jawab men-serialize struct ini
// ke HTTP response menggunakan framework pilihan (Fiber, net/http, dll).
//
// Format sukses:   {"data": ..., "meta": ...}
// Format error:    {"error": {"code": "...", "message": "...", "details": ..., "request_id": "..."}}
// Format paginasi: {"data": [...], "meta": {"pagination": {"limit": N, "next_cursor": "...", "has_more": bool}}}
//
// Rujukan: TDD §5.1, §3.7; Roadmap Fase 1 §4.4; SRS §7.1.
package response

import (
	"strings"

	"github.com/wesign/wesign-backend/internal/shared/errorx"
)

// Response adalah amplop standar untuk respon sukses.
type Response struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

// ErrorBody adalah isi dari key "error" pada amplop error.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse adalah amplop standar untuk respon error.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// PaginationMeta berisi metadata cursor-based pagination (TDD §3.7).
type PaginationMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewSuccess membuat amplop sukses tanpa meta.
func NewSuccess(data any) Response {
	return Response{Data: data}
}

// NewSuccessWithMeta membuat amplop sukses dengan metadata.
func NewSuccessWithMeta(data, meta any) Response {
	return Response{Data: data, Meta: meta}
}

// Paginated membuat amplop sukses dengan metadata pagination cursor-based.
func Paginated(data any, limit int, nextCursor string, hasMore bool) Response {
	return Response{
		Data: data,
		Meta: map[string]any{
			"pagination": PaginationMeta{
				Limit:      limit,
				NextCursor: nextCursor,
				HasMore:    hasMore,
			},
		},
	}
}

// NewError membuat amplop error secara manual.
// CATATAN KEAMANAN: details dikirim apa adanya tanpa sanitasi.
// Gunakan FromAppError untuk sanitasi otomatis kunci sensitif.
func NewError(code, message, requestID string, details any) ErrorResponse {
	return ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
	}
}

// FromAppError mengonversi *errorx.AppError menjadi ErrorResponse yang aman
// untuk dikirim ke klien. Kunci sensitif di Details disanitasi otomatis.
func FromAppError(appErr *errorx.AppError, requestID string) ErrorResponse {
	if appErr == nil {
		return NewError(
			errorx.CodeInternal,
			"Terjadi kesalahan internal",
			requestID,
			nil,
		)
	}

	return ErrorResponse{
		Error: ErrorBody{
			Code:      appErr.Code,
			Message:   appErr.Message,
			Details:   SanitizeDetails(appErr.Details),
			RequestID: requestID,
		},
	}
}

// sensitiveDetailKeySubstrings adalah daftar substring (case-insensitive)
// yang menandakan kunci Detail bersifat internal/sensitif dan tidak boleh
// bocor ke klien.
var sensitiveDetailKeySubstrings = []string{
	"internal",
	"stack_trace",
	"stacktrace",
	"raw_error",
	"sql_error",
	"db_error",
	"debug_info",
	"connection_string",
	"dsn",
}

// IsSensitiveDetailKey memeriksa apakah sebuah kunci mengandung substring sensitif.
func IsSensitiveDetailKey(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range sensitiveDetailKeySubstrings {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SanitizeDetails menghapus kunci sensitif dari Details secara rekursif.
// Mengembalikan nil jika semua kunci di dalamnya sensitif (agar omitted dari JSON).
func SanitizeDetails(details any) any {
	switch v := details.(type) {
	case map[string]string:
		return sanitizeStringMap(v)
	case map[string]any:
		return sanitizeAnyMap(v)
	case []any:
		return sanitizeSlice(v)
	default:
		return details
	}
}

func sanitizeStringMap(v map[string]string) any {
	result := make(map[string]string, len(v))
	for k, val := range v {
		if IsSensitiveDetailKey(k) {
			continue
		}
		result[k] = val
	}
	return nilIfEmptyStringMap(result)
}

func sanitizeAnyMap(v map[string]any) any {
	result := make(map[string]any, len(v))
	for k, val := range v {
		if IsSensitiveDetailKey(k) {
			continue
		}
		result[k] = SanitizeDetails(val)
	}
	return nilIfEmptyAnyMap(result)
}

func sanitizeSlice(v []any) []any {
	result := make([]any, 0, len(v))
	for _, item := range v {
		result = append(result, SanitizeDetails(item))
	}
	return result
}

func nilIfEmptyStringMap(m map[string]string) any {
	if len(m) == 0 {
		return nil
	}
	return m
}

func nilIfEmptyAnyMap(m map[string]any) any {
	if len(m) == 0 {
		return nil
	}
	return m
}
