// Package errorx menyediakan tipe error domain terpusat (AppError) beserta
// registry kode error yang selaras dengan kontrak TDD §5.1–§5.10 dan SRS §10.2.
//
// Prinsip desain:
//   - Satu format error untuk seluruh API: {code, message, details, request_id}
//     (TDD §5.1). Penyisipan request_id dilakukan di pkg/response.
//   - InternalErr hanya dipakai untuk logging dan tidak pernah diserialisasi
//     ke response (SRS §10.3: stack trace hanya di sisi server).
//   - Immutable: WithDetails/WithDetail/WithMessage mengembalikan salinan
//     sehingga satu error sentinel aman dipakai bersama banyak request.
package errorx

import (
	"errors"
	"fmt"
	"net/http"
)

// ─ Registry kode error (TDD §5.1–§5.10, SRS §10.2) ────────────────

const (
	// Umum & transport (TDD §5.1).
	CodeValidation         = "VALIDATION_ERROR"
	CodeUnprocessable      = "UNPROCESSABLE_ENTITY"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodePayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	CodeRateLimited        = "RATE_LIMIT_EXCEEDED"
	CodeInternal           = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"

	// Perlindungan keamanan (TDD §8.4 CSRF, §8.9 MFA).
	CodeCSRFInvalid   = "CSRF_INVALID"
	CodeMFARequired   = "MFA_REQUIRED"
	CodeMFAInvalid    = "MFA_INVALID"
	CodeTurnstileFail = "TURNSTILE_FAILED"

	// Autentikasi & identitas (TDD §5.2, §8.1).
	CodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	CodeAuthTokenInvalid       = "AUTH_TOKEN_INVALID"
	CodeAuthTokenExpired       = "AUTH_TOKEN_EXPIRED"
	CodeAuthTokenReused        = "AUTH_TOKEN_REUSED"
	CodeAuthMissingToken       = "AUTH_MISSING_TOKEN"
	CodeAuthInvalidFormat      = "AUTH_INVALID_FORMAT"
	CodeAuthWrongTokenType     = "AUTH_WRONG_TOKEN_TYPE"
	CodeAuthAccountLocked      = "AUTH_ACCOUNT_LOCKED"
	CodeAuthForbidden          = "AUTH_FORBIDDEN"
	CodeAuthEmailNotVerified   = "AUTH_EMAIL_NOT_VERIFIED"
	CodeEmailAlreadyExists     = "EMAIL_ALREADY_EXISTS"

	// Dokumen (TDD §5.3, SRS §10.2).
	CodeDocumentInvalidFormat = "DOCUMENT_INVALID_FORMAT"
	CodeDocumentTooLarge      = "DOCUMENT_TOO_LARGE"
	CodeDocumentDuplicate     = "DOCUMENT_DUPLICATE"
	CodeDocumentNotFound      = "DOCUMENT_NOT_FOUND"
	CodeDocumentAlreadySigned = "DOCUMENT_ALREADY_SIGNED"
	CodeDocumentEncrypted     = "DOCUMENT_ENCRYPTED"
	CodeDocumentHasJavaScript = "DOCUMENT_HAS_JAVASCRIPT"
	CodeMalwareDetected       = "MALWARE_DETECTED"
	CodePlacementOutOfBounds  = "PLACEMENT_OUT_OF_BOUNDS"
	CodePlacementOverlap      = "PLACEMENT_OVERLAP"
	CodeInvalidPageNumber     = "INVALID_PAGE_NUMBER"

	// Package signing (TDD §5.4).
	CodePackageMaxDocuments  = "PACKAGE_MAX_DOCUMENTS"
	CodePackageEmpty         = "PACKAGE_EMPTY"
	CodePackageAlreadyActive = "PACKAGE_ALREADY_ACTIVE"
	CodeIncompleteDocuments  = "INCOMPLETE_DOCUMENTS"

	// Penandatanganan & multi-signer (TDD §5.5).
	CodeSigningOTPInvalid             = "SIGNING_OTP_INVALID"
	CodeSigningOTPExpired             = "SIGNING_OTP_EXPIRED"
	CodeSigningOTPMaxAttempts         = "SIGNING_OTP_MAX_ATTEMPTS"
	CodeSigningRequestExpired         = "SIGNING_REQUEST_EXPIRED"
	CodeSigningRequestAlreadySigned   = "SIGNING_REQUEST_ALREADY_SIGNED"
	CodeSigningRequestAlreadyDeclined = "SIGNING_REQUEST_ALREADY_DECLINED"
	CodeSigningSequenceInvalid        = "SIGNING_SEQUENCE_INVALID"
	CodeSigningMaxSigners             = "SIGNING_MAX_SIGNERS"
	CodeSigningRequestNotRemovable    = "SIGNING_REQUEST_NOT_REMOVABLE"
	CodeDuplicateSignerEmail          = "DUPLICATE_SIGNER_EMAIL"

	// Verifikasi publik (TDD §5.6).
	CodeVerifyNotRegistered = "VERIFY_NOT_REGISTERED"
	CodeVerifyLinkExpired   = "VERIFY_LINK_EXPIRED"

	// Aset tanda tangan (TDD §5.7).
	CodeFileTooLarge      = "FILE_TOO_LARGE"
	CodeInvalidFileFormat = "INVALID_FILE_FORMAT"

	// Background job (TDD §5.8).
	CodeJobNotFound = "JOB_NOT_FOUND"
)

// statusByCode memetakan kode error ke HTTP status resmi (TDD §5.1–§5.10).
var statusByCode = map[string]int{
	CodeValidation:         http.StatusBadRequest,
	CodeUnprocessable:      http.StatusUnprocessableEntity,
	CodeNotFound:           http.StatusNotFound,
	CodeConflict:           http.StatusConflict,
	CodePayloadTooLarge:    http.StatusRequestEntityTooLarge,
	CodeRateLimited:        http.StatusTooManyRequests,
	CodeInternal:           http.StatusInternalServerError,
	CodeServiceUnavailable: http.StatusServiceUnavailable,

	CodeCSRFInvalid:   http.StatusForbidden,
	CodeMFARequired:   http.StatusUnauthorized,
	CodeMFAInvalid:    http.StatusUnauthorized,
	CodeTurnstileFail: http.StatusUnprocessableEntity,

	CodeAuthInvalidCredentials: http.StatusUnauthorized,
	CodeAuthTokenInvalid:       http.StatusUnauthorized,
	CodeAuthTokenExpired:       http.StatusUnauthorized,
	CodeAuthTokenReused:        http.StatusUnauthorized,
	CodeAuthMissingToken:       http.StatusUnauthorized,
	CodeAuthInvalidFormat:      http.StatusUnauthorized,
	CodeAuthWrongTokenType:     http.StatusUnauthorized,
	CodeAuthAccountLocked:      http.StatusLocked,
	CodeAuthForbidden:          http.StatusForbidden,
	CodeAuthEmailNotVerified:   http.StatusForbidden,
	CodeEmailAlreadyExists:     http.StatusConflict,

	CodeDocumentInvalidFormat: http.StatusUnprocessableEntity,
	CodeDocumentTooLarge:      http.StatusRequestEntityTooLarge,
	CodeDocumentDuplicate:     http.StatusConflict,
	CodeDocumentNotFound:      http.StatusNotFound,
	CodeDocumentAlreadySigned: http.StatusConflict,
	CodeDocumentEncrypted:     http.StatusUnprocessableEntity,
	CodeDocumentHasJavaScript: http.StatusUnprocessableEntity,
	CodeMalwareDetected:       http.StatusUnprocessableEntity,
	CodePlacementOutOfBounds:  http.StatusUnprocessableEntity,
	CodePlacementOverlap:      http.StatusUnprocessableEntity,
	CodeInvalidPageNumber:     http.StatusUnprocessableEntity,

	CodePackageMaxDocuments:  http.StatusUnprocessableEntity,
	CodePackageEmpty:         http.StatusUnprocessableEntity,
	CodePackageAlreadyActive: http.StatusConflict,
	CodeIncompleteDocuments:  http.StatusUnprocessableEntity,

	CodeSigningOTPInvalid:             http.StatusUnauthorized,
	CodeSigningOTPExpired:             http.StatusUnauthorized,
	CodeSigningOTPMaxAttempts:         http.StatusTooManyRequests,
	CodeSigningRequestExpired:         http.StatusGone,
	CodeSigningRequestAlreadySigned:   http.StatusConflict,
	CodeSigningRequestAlreadyDeclined: http.StatusConflict,
	CodeSigningSequenceInvalid:        http.StatusConflict,
	CodeSigningMaxSigners:             http.StatusUnprocessableEntity,
	CodeSigningRequestNotRemovable:    http.StatusConflict,
	CodeDuplicateSignerEmail:          http.StatusUnprocessableEntity,

	CodeVerifyNotRegistered: http.StatusNotFound,
	CodeVerifyLinkExpired:   http.StatusGone,

	CodeFileTooLarge:      http.StatusUnprocessableEntity,
	CodeInvalidFileFormat: http.StatusUnprocessableEntity,

	CodeJobNotFound: http.StatusNotFound,
}

// StatusFor mengembalikan HTTP status untuk sebuah kode error.
// Kode yang tidak terdaftar diperlakukan sebagai INTERNAL_ERROR (500).
func StatusFor(code string) int {
	if status, ok := statusByCode[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// ─ Tipe error domain ──────────────────────────────────────────────

// AppError adalah satu-satunya tipe error yang boleh keluar dari layer
// use case dan handler. Bentuk field-nya mengikuti amplop error TDD §5.1.
type AppError struct {
	Code        string         // kode mesin, mis. DOCUMENT_DUPLICATE
	Message     string         // pesan untuk pengguna
	HTTPStatus  int            // status HTTP hasil pemetaan kode
	Details     map[string]any // informasi tambahan, mis. {"field": "file"}
	InternalErr error          // error asli untuk logging; TIDAK diserialisasi
}

// New membuat AppError dengan HTTP status yang diambil dari registry kode.
// Untuk kode yang tidak terdaftar, status menjadi 500 INTERNAL_ERROR.
func New(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: StatusFor(code),
	}
}

// Newf membuat AppError dengan pesan berformat.
func Newf(code, format string, args ...any) *AppError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap membungkus error internal ke dalam AppError tanpa mengubah kode.
// InternalErr hanya muncul di log (lihat Error), bukan di response HTTP.
func Wrap(code, message string, internal error) *AppError {
	err := New(code, message)
	err.InternalErr = internal
	return err
}

// Error memenuhi interface error. Pesan yang dihasilkan dipakai untuk log,
// sehingga error internal ikut disertakan untuk keperluan diagnosis.
func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.InternalErr != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.InternalErr)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap memungkinkan errors.Is/errors.As menelusuri error internal.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.InternalErr
}

// Is membandingkan berdasarkan Code sehingga errors.Is(err, errorx.ErrNotFound)
// tetap bekerja walau pesan atau details berbeda.
func (e *AppError) Is(target error) bool {
	if e == nil {
		return false
	}
	t, ok := target.(*AppError)
	return ok && e.Code == t.Code
}

// WithDetails mengembalikan salinan AppError dengan details gabungan.
// Nilai asli tidak berubah (immutable) sehingga aman untuk error sentinel.
func (e *AppError) WithDetails(details map[string]any) *AppError {
	if e == nil {
		return nil
	}
	if len(details) == 0 {
		return e.clone()
	}
	merged := make(map[string]any, len(e.Details)+len(details))
	for key, value := range e.Details {
		merged[key] = value
	}
	for key, value := range details {
		merged[key] = value
	}
	clone := e.clone()
	clone.Details = merged
	return clone
}

// WithDetail menambahkan satu pasangan key-value pada details.
func (e *AppError) WithDetail(key string, value any) *AppError {
	return e.WithDetails(map[string]any{key: value})
}

// WithField menambahkan detail {"field": field, "reason": reason} yang dipakai
// secara konsisten oleh validator (TDD §5.1) dan use case.
func (e *AppError) WithField(field, reason string) *AppError {
	return e.WithDetails(map[string]any{"field": field, "reason": reason})
}

// WithMessage mengganti pesan pengguna tanpa mengubah tipe aslinya.
func (e *AppError) WithMessage(message string) *AppError {
	if e == nil {
		return nil
	}
	clone := e.clone()
	clone.Message = message
	return clone
}

// clone menyalin AppError beserta map details agar tidak terjadi aliasing.
func (e *AppError) clone() *AppError {
	clone := *e
	if len(e.Details) > 0 {
		details := make(map[string]any, len(e.Details))
		for key, value := range e.Details {
			details[key] = value
		}
		clone.Details = details
	}
	return &clone
}

// As mengambil *AppError dari rantai error; nil jika bukan AppError.
func As(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// HasCode memeriksa kode error tanpa perlu menyediakan target sentinel.
func HasCode(err error, code string) bool {
	appErr := As(err)
	return appErr != nil && appErr.Code == code
}

// ─ Konstruktor umum (roadmap Fase 1 §4.2) ─────────────────────────

// Validation → 400 VALIDATION_ERROR, dipakai ketika payload tidak lolos binding.
func Validation(message string) *AppError { return New(CodeValidation, message) }

// Unprocessable → 422, dipakai ketika validasi bisnis gagal (TDD §5.1).
func Unprocessable(message string) *AppError { return New(CodeUnprocessable, message) }

// NotFound → 404.
func NotFound(message string) *AppError { return New(CodeNotFound, message) }

// Conflict → 409, mis. konflik state atau unique constraint.
func Conflict(message string) *AppError { return New(CodeConflict, message) }

// TooLarge → 413, mis. DOCUMENT_TOO_LARGE (max 25 MB, FR-DOC-01).
func TooLarge(message string) *AppError { return New(CodePayloadTooLarge, message) }

// RateLimited → 429.
func RateLimited(message string) *AppError { return New(CodeRateLimited, message) }

// Unavailable → 503, dipakai saat dependensi eksternal tidak siap.
func Unavailable(message string) *AppError { return New(CodeServiceUnavailable, message) }

// Unauthorized → 401 AUTH_TOKEN_INVALID, default kegagalan autentikasi.
func Unauthorized(message string) *AppError { return New(CodeAuthTokenInvalid, message) }

// UnauthorizedWith → 401 dengan kode spesifik seperti AUTH_INVALID_CREDENTIALS,
// AUTH_ACCOUNT_LOCKED, atau SIGNING_OTP_INVALID.
func UnauthorizedWith(code, message string) *AppError { return New(code, message) }

// Forbidden → 403 AUTH_FORBIDDEN (RBAC dan pemeriksaan kepemilikan resource).
func Forbidden(message string) *AppError { return New(CodeAuthForbidden, message) }

// Internal → 500 INTERNAL_ERROR.
func Internal(message string) *AppError { return New(CodeInternal, message) }

// InternalWrap → 500 INTERNAL_ERROR sambil menyimpan error asli untuk log.
func InternalWrap(message string, internal error) *AppError {
	return Wrap(CodeInternal, message, internal)
}

// ─ Error sentinel lintas modul ────────────────────────────────────

var (
	// ErrNotFound dipakai ketika resource tidak ada.
	ErrNotFound = NotFound("Sumber daya tidak ditemukan")
	// ErrForbidden dipakai ketika kepemilikan resource tidak sesuai.
	ErrForbidden = Forbidden("Anda tidak memiliki akses ke sumber daya ini")
	// ErrUnauthorized dipakai ketika token tidak ada atau tidak valid.
	ErrUnauthorized = Unauthorized("Autentikasi diperlukan")
	// ErrInternal dipakai untuk kegagalan tak terduga.
	ErrInternal = Internal("Terjadi kesalahan internal")
	// ErrInvalidCredentials → FR-AUTH-02 (rate limit 5 gagal / 15 menit).
	ErrInvalidCredentials = New(CodeAuthInvalidCredentials, "Email atau kata sandi salah")
	// ErrEmailNotVerified → FR-AUTH-02 (akun pending tidak boleh login).
	ErrEmailNotVerified = New(CodeAuthEmailNotVerified, "Email belum diverifikasi")
	// ErrAccountLocked → FR-AUTH-02 (lock 30 menit).
	ErrAccountLocked = New(CodeAuthAccountLocked, "Akun terkunci sementara")
	// ErrRateLimited → per-IP dan per-user (TDD §8.3).
	ErrRateLimited = RateLimited("Terlalu banyak permintaan. Coba lagi nanti.")
	// ErrTokenExpired → FR-AUTH-05.
	ErrTokenExpired = New(CodeAuthTokenExpired, "Token kedaluwarsa")
	// ErrTokenReused → FR-AUTH-05, memicu pencabutan seluruh token family.
	ErrTokenReused = New(CodeAuthTokenReused, "Token telah digunakan dan dicabut")
)
