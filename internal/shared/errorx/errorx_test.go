package errorx_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wesign/wesign-backend/internal/shared/errorx"
)

func TestStatusFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		code     string
		expected int
	}{
		{"validation 400", errorx.CodeValidation, http.StatusBadRequest},
		{"unprocessable 422", errorx.CodeUnprocessable, http.StatusUnprocessableEntity},
		{"not found 404", errorx.CodeNotFound, http.StatusNotFound},
		{"conflict 409", errorx.CodeConflict, http.StatusConflict},
		{"payload too large 413", errorx.CodePayloadTooLarge, http.StatusRequestEntityTooLarge},
		{"rate limited 429", errorx.CodeRateLimited, http.StatusTooManyRequests},
		{"internal 500", errorx.CodeInternal, http.StatusInternalServerError},
		{"unavailable 503", errorx.CodeServiceUnavailable, http.StatusServiceUnavailable},
		{"account locked 423", errorx.CodeAuthAccountLocked, http.StatusLocked},
		{"csrf 403", errorx.CodeCSRFInvalid, http.StatusForbidden},
		{"turnstile 422", errorx.CodeTurnstileFail, http.StatusUnprocessableEntity},
		{"otp max attempts 429", errorx.CodeSigningOTPMaxAttempts, http.StatusTooManyRequests},
		{"signing request expired 410", errorx.CodeSigningRequestExpired, http.StatusGone},
		{"verify link expired 410", errorx.CodeVerifyLinkExpired, http.StatusGone},
		{"document duplicate 409", errorx.CodeDocumentDuplicate, http.StatusConflict},
		{"job not found 404", errorx.CodeJobNotFound, http.StatusNotFound},
		{"kode tidak terdaftar -> 500", "CODE_YANG_TIDAK_ADA", http.StatusInternalServerError},
		{"kode kosong -> 500", "", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, errorx.StatusFor(tc.code))
		})
	}
}

func TestNew_UsesRegistryForHTTPStatus(t *testing.T) {
	t.Parallel()

	err := errorx.New(errorx.CodeDocumentDuplicate, "Dokumen duplikat")

	require.NotNil(t, err)
	assert.Equal(t, errorx.CodeDocumentDuplicate, err.Code)
	assert.Equal(t, "Dokumen duplikat", err.Message)
	assert.Equal(t, http.StatusConflict, err.HTTPStatus)
	assert.Nil(t, err.Details)
	assert.NoError(t, err.InternalErr)
}

func TestNew_UnknownCodeFallsBackToInternal(t *testing.T) {
	t.Parallel()

	err := errorx.New("TIDAK_DIKENAL", "pesan")

	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
}

func TestNewf_FormatsMessage(t *testing.T) {
	t.Parallel()

	err := errorx.Newf(errorx.CodeValidation, "field %s wajib diisi", "email")

	assert.Equal(t, "field email wajib diisi", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
}

func TestWrap_KeepsInternalError(t *testing.T) {
	t.Parallel()

	root := errors.New("connection refused")
	err := errorx.Wrap(errorx.CodeInternal, "Gagal menyimpan dokumen", root)

	assert.Equal(t, root, err.InternalErr)
	assert.ErrorIs(t, err, root, "Unwrap harus menelusuri error internal")
	assert.NotContains(t, err.Details, "internal", "error internal tidak boleh masuk details")
	assert.Contains(t, err.Error(), "connection refused", "Error() dipakai untuk log")
}

func TestError_Format(t *testing.T) {
	t.Parallel()

	t.Run("tanpa internal error", func(t *testing.T) {
		t.Parallel()
		err := errorx.New(errorx.CodeNotFound, "Dokumen tidak ditemukan")
		assert.Equal(t, "NOT_FOUND: Dokumen tidak ditemukan", err.Error())
	})

	t.Run("dengan internal error", func(t *testing.T) {
		t.Parallel()
		err := errorx.InternalWrap("Gagal", errors.New("boom"))
		assert.Equal(t, "INTERNAL_ERROR: Gagal: boom", err.Error())
	})

	t.Run("receiver nil", func(t *testing.T) {
		t.Parallel()
		var err *errorx.AppError
		assert.Equal(t, "<nil>", err.Error())
		assert.NoError(t, err.Unwrap())
		assert.Nil(t, err.WithDetails(nil))
		assert.Nil(t, err.WithMessage("x"))
		assert.False(t, err.Is(errorx.ErrNotFound))
	})
}

func TestIs_ComparesByCode(t *testing.T) {
	t.Parallel()

	// Pesan berbeda, kode sama -> tetap dianggap error yang sama.
	other := errorx.New(errorx.CodeNotFound, "pesan lain")
	assert.ErrorIs(t, other, errorx.ErrNotFound)
	assert.ErrorIs(t, errorx.ErrNotFound.WithDetail("id", "abc"), errorx.ErrNotFound)

	// Kode berbeda -> bukan error yang sama.
	assert.NotErrorIs(t, errorx.ErrForbidden, errorx.ErrNotFound)

	// Target bukan AppError -> false (tidak panic).
	assert.False(t, errorx.ErrNotFound.Is(errors.New("plain")))
}

func TestAsAndHasCode(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("handler: %w", errorx.ErrForbidden)

	appErr := errorx.As(wrapped)
	require.NotNil(t, appErr)
	assert.Equal(t, errorx.CodeAuthForbidden, appErr.Code)

	assert.True(t, errorx.HasCode(wrapped, errorx.CodeAuthForbidden))
	assert.False(t, errorx.HasCode(wrapped, errorx.CodeNotFound))
	assert.Nil(t, errorx.As(errors.New("plain error")))
	assert.False(t, errorx.HasCode(errors.New("plain error"), errorx.CodeNotFound))
}

// ─ Immutability (roadmap Fase 1 §4.2) ─────────────────────────────

func TestWithDetails_IsImmutable(t *testing.T) {
	t.Parallel()

	base := errorx.ErrNotFound.WithDetails(map[string]any{"resource": "document"})
	derived := base.WithDetails(map[string]any{"id": "doc-1"})

	assert.NotSame(t, base, derived, "WithDetails harus mengembalikan salinan")
	assert.Len(t, base.Details, 1)
	assert.Len(t, derived.Details, 2)
	assert.Equal(t, "document", derived.Details["resource"])
	assert.Equal(t, "doc-1", derived.Details["id"])

	// Memutasi details hasil salinan tidak boleh memengaruhi nilai asal.
	derived.Details["injected"] = true
	assert.NotContains(t, base.Details, "injected")
	assert.NotContains(t, errorx.ErrNotFound.Details, "injected")
	assert.Nil(t, errorx.ErrNotFound.Details, "sentinel global tidak boleh tercemar")
}

func TestWithDetails_EmptyReturnsCopy(t *testing.T) {
	t.Parallel()

	base := errorx.ErrForbidden.WithDetail("scope", "document")
	same := base.WithDetails(nil)

	assert.NotSame(t, base, same)
	assert.Equal(t, base.Details, same.Details)
}

func TestWithDetailAndWithField(t *testing.T) {
	t.Parallel()

	t.Run("WithDetail", func(t *testing.T) {
		t.Parallel()
		err := errorx.Validation("payload tidak valid").WithDetail("field", "file")
		assert.Equal(t, "file", err.Details["field"])
	})

	t.Run("WithField menambah field dan reason", func(t *testing.T) {
		t.Parallel()
		err := errorx.New(errorx.CodeDocumentInvalidFormat, "Bukan PDF").
			WithField("file", "magic_bytes_mismatch")

		assert.Equal(t, "file", err.Details["field"])
		assert.Equal(t, "magic_bytes_mismatch", err.Details["reason"])

		// Rantai pemanggilan tetap menjaga details sebelumnya.
		err = err.WithDetail("size", 1024)
		assert.Equal(t, "file", err.Details["field"])
		assert.Equal(t, 1024, err.Details["size"])
	})
}

func TestWithMessage_IsImmutable(t *testing.T) {
	t.Parallel()

	base := errorx.ErrNotFound
	derived := base.WithMessage("Dokumen tidak ditemukan")

	assert.NotSame(t, base, derived)
	assert.Equal(t, "Sumber daya tidak ditemukan", base.Message)
	assert.Equal(t, "Dokumen tidak ditemukan", derived.Message)
	assert.Equal(t, base.HTTPStatus, derived.HTTPStatus, "status tidak berubah")
	assert.Equal(t, base.Code, derived.Code, "kode tidak berubah")
}

// ─ Konstruktor & sentinel ─────────────────────────────────────────

func TestConstructors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		err    *errorx.AppError
		code   string
		status int
	}{
		{"Validation", errorx.Validation("x"), errorx.CodeValidation, http.StatusBadRequest},
		{"Unprocessable", errorx.Unprocessable("x"), errorx.CodeUnprocessable, http.StatusUnprocessableEntity},
		{"NotFound", errorx.NotFound("x"), errorx.CodeNotFound, http.StatusNotFound},
		{"Conflict", errorx.Conflict("x"), errorx.CodeConflict, http.StatusConflict},
		{"TooLarge", errorx.TooLarge("x"), errorx.CodePayloadTooLarge, http.StatusRequestEntityTooLarge},
		{"RateLimited", errorx.RateLimited("x"), errorx.CodeRateLimited, http.StatusTooManyRequests},
		{"Unavailable", errorx.Unavailable("x"), errorx.CodeServiceUnavailable, http.StatusServiceUnavailable},
		{"Unauthorized", errorx.Unauthorized("x"), errorx.CodeAuthTokenInvalid, http.StatusUnauthorized},
		{"UnauthorizedWith", errorx.UnauthorizedWith(errorx.CodeSigningOTPInvalid, "x"), errorx.CodeSigningOTPInvalid, http.StatusUnauthorized},
		{"Forbidden", errorx.Forbidden("x"), errorx.CodeAuthForbidden, http.StatusForbidden},
		{"Internal", errorx.Internal("x"), errorx.CodeInternal, http.StatusInternalServerError},
		{"InternalWrap", errorx.InternalWrap("x", errors.New("boom")), errorx.CodeInternal, http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, tc.err)
			assert.Equal(t, tc.code, tc.err.Code)
			assert.Equal(t, tc.status, tc.err.HTTPStatus)
			assert.Equal(t, "x", tc.err.Message)
		})
	}
}

func TestSentinels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{"ErrNotFound", errorx.ErrNotFound, errorx.CodeNotFound, http.StatusNotFound},
		{"ErrForbidden", errorx.ErrForbidden, errorx.CodeAuthForbidden, http.StatusForbidden},
		{"ErrUnauthorized", errorx.ErrUnauthorized, errorx.CodeAuthTokenInvalid, http.StatusUnauthorized},
		{"ErrInternal", errorx.ErrInternal, errorx.CodeInternal, http.StatusInternalServerError},
		{"ErrInvalidCredentials", errorx.ErrInvalidCredentials, errorx.CodeAuthInvalidCredentials, http.StatusUnauthorized},
		{"ErrEmailNotVerified", errorx.ErrEmailNotVerified, errorx.CodeAuthEmailNotVerified, http.StatusForbidden},
		{"ErrAccountLocked", errorx.ErrAccountLocked, errorx.CodeAuthAccountLocked, http.StatusLocked},
		{"ErrRateLimited", errorx.ErrRateLimited, errorx.CodeRateLimited, http.StatusTooManyRequests},
		{"ErrTokenExpired", errorx.ErrTokenExpired, errorx.CodeAuthTokenExpired, http.StatusUnauthorized},
		{"ErrTokenReused", errorx.ErrTokenReused, errorx.CodeAuthTokenReused, http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.err, errorx.New(tc.code, "pesan apa pun"))
			assert.True(t, errorx.HasCode(tc.err, tc.code))
			assert.Equal(t, tc.status, errorx.As(tc.err).HTTPStatus)
		})
	}
}

// TestRegistry_AllCodesRegistered memastikan setiap kode yang dideklarasikan
// pada paket ini benar-benar memiliki pemetaan HTTP status (mencegah typo pada
// registry yang hanya akan terlihat saat runtime).
func TestRegistry_AllCodesRegistered(t *testing.T) {
	t.Parallel()

	codes := []string{
		errorx.CodeValidation, errorx.CodeUnprocessable, errorx.CodeNotFound,
		errorx.CodeConflict, errorx.CodePayloadTooLarge, errorx.CodeRateLimited,
		errorx.CodeServiceUnavailable,
		errorx.CodeCSRFInvalid, errorx.CodeMFARequired, errorx.CodeMFAInvalid,
		errorx.CodeTurnstileFail,
		errorx.CodeAuthInvalidCredentials, errorx.CodeAuthTokenInvalid,
		errorx.CodeAuthTokenExpired, errorx.CodeAuthTokenReused,
		errorx.CodeAuthMissingToken, errorx.CodeAuthInvalidFormat,
		errorx.CodeAuthWrongTokenType, errorx.CodeAuthAccountLocked,
		errorx.CodeAuthForbidden, errorx.CodeAuthEmailNotVerified,
		errorx.CodeEmailAlreadyExists,
		errorx.CodeDocumentInvalidFormat, errorx.CodeDocumentTooLarge,
		errorx.CodeDocumentDuplicate, errorx.CodeDocumentNotFound,
		errorx.CodeDocumentAlreadySigned, errorx.CodeDocumentEncrypted,
		errorx.CodeDocumentHasJavaScript, errorx.CodeMalwareDetected,
		errorx.CodePlacementOutOfBounds, errorx.CodePlacementOverlap,
		errorx.CodeInvalidPageNumber,
		errorx.CodePackageMaxDocuments, errorx.CodePackageEmpty,
		errorx.CodePackageAlreadyActive, errorx.CodeIncompleteDocuments,
		errorx.CodeSigningOTPInvalid, errorx.CodeSigningOTPExpired,
		errorx.CodeSigningOTPMaxAttempts, errorx.CodeSigningRequestExpired,
		errorx.CodeSigningRequestAlreadySigned, errorx.CodeSigningRequestAlreadyDeclined,
		errorx.CodeSigningSequenceInvalid, errorx.CodeSigningMaxSigners,
		errorx.CodeSigningRequestNotRemovable, errorx.CodeDuplicateSignerEmail,
		errorx.CodeVerifyNotRegistered, errorx.CodeVerifyLinkExpired,
		errorx.CodeFileTooLarge, errorx.CodeInvalidFileFormat,
		errorx.CodeJobNotFound,
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			assert.NotEqual(t, http.StatusInternalServerError, errorx.StatusFor(code),
				"kode %s belum terdaftar pada registry", code)
			assert.NotEmpty(t, errorx.New(code, "pesan").Error())
		})
	}

	// INTERNAL_ERROR memang 500 dan tetap harus terdaftar eksplisit.
	assert.Equal(t, http.StatusInternalServerError, errorx.StatusFor(errorx.CodeInternal))
}
