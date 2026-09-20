package validator_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wesign/wesign-backend/internal/shared/errorx"
	"github.com/wesign/wesign-backend/internal/shared/validator"
)

// sampleDTO mendefinisikan tag validasi yang mirur kontrak DTO real pada
// TDD §5.2 (register): email, password (min/max), role (oneof), doc_id (uuid).
type sampleDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=32"`
	Role     string `json:"role" validate:"required,oneof=admin user"`
	DocID    string `json:"doc_id" validate:"omitempty,uuid"`
	Code     string `json:"code" validate:"omitempty,numeric"`
}

func TestValidator_ValidStruct(t *testing.T) {
	t.Parallel()

	dto := sampleDTO{
		Email:    "test@wesign.id",
		Password: "password123",
		Role:     "user",
		DocID:    "550e8400-e29b-41d4-a716-446655440000",
		Code:     "123456",
	}

	assert.NoError(t, validator.ValidateStruct(dto))
}

// TestValidator_InvalidStruct_MultiField memverifikasi pemetaan ke amplop
// TDD §5.1: 422 UNPROCESSABLE_ENTITY + details key memakai JSON tag snake_case.
func TestValidator_InvalidStruct_MultiField(t *testing.T) {
	t.Parallel()

	dto := sampleDTO{
		Email:    "not-an-email",
		Password: "this_password_is_way_too_long_and_exceeds_thirty_two_characters_limit",
		Role:     "superadmin",
		DocID:    "not-a-uuid",
		Code:     "abc",
	}

	err := validator.ValidateStruct(dto)
	require.Error(t, err)

	appErr := errorx.As(err)
	require.NotNil(t, appErr)
	assert.Equal(t, errorx.CodeUnprocessable, appErr.Code)
	assert.Equal(t, http.StatusUnprocessableEntity, appErr.HTTPStatus)

	details := appErr.Details
	require.NotNil(t, details)
	assert.Contains(t, details, "email")
	assert.Contains(t, details, "password")
	assert.Contains(t, details, "role")
	assert.Contains(t, details, "doc_id")
	assert.Contains(t, details, "code")

	// TIDAK boleh memakai nama field Go (jembatan OpenAPI, TDD §5.1).
	assert.NotContains(t, details, "docid")
	assert.NotContains(t, details, "docidlower")

	// Pesan per tag.
	assert.Equal(t, "Field 'email' harus berupa alamat email yang valid", details["email"].(string))
	assert.Equal(t, "Field 'password' maksimal 32 karakter/nilai", details["password"].(string))
	assert.Equal(t, "Field 'role' harus salah satu dari: admin user", details["role"].(string))
	assert.Equal(t, "Field 'doc_id' harus berupa format UUID yang valid", details["doc_id"].(string))
	assert.Equal(t, "Field 'code' harus berupa angka", details["code"].(string))
}

// TestValidator_NonStruct memverifikasi jalur input bukan struct:
// TDD §5.1 → 400 VALIDATION_ERROR ("request body tidak valid").
func TestValidator_NonStruct(t *testing.T) {
	t.Parallel()

	err := validator.ValidateStruct("not-a-struct")
	require.Error(t, err)

	appErr := errorx.As(err)
	require.NotNil(t, appErr)
	assert.Equal(t, errorx.CodeValidation, appErr.Code)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
}

// Fallback ke nama field Go ketika tidak ada JSON tag.
type noTagDTO struct {
	PlainField string `validate:"required"`
}

func TestValidator_FallbackToFieldNameWhenNoJSONTag(t *testing.T) {
	t.Parallel()

	err := validator.ValidateStruct(noTagDTO{})
	require.Error(t, err)

	appErr := errorx.As(err)
	require.NotNil(t, appErr)

	details := appErr.Details
	require.NotNil(t, details)
	assert.Contains(t, details, "PlainField", "tanpa JSON tag, nama field Go dipakai")
	assert.Equal(t, "Field 'PlainField' wajib diisi", details["PlainField"].(string))
}

// TestValidator_RequiredAndLengthTags memverifikasi tag min/len/gte yang
// dipakai pada kontrak register TDD §5.2.
type lengthDTO struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
	Pin  string `json:"pin" validate:"omitempty,len=6"`
}

func TestValidator_RequiredAndLengthTags(t *testing.T) {
	t.Parallel()

	t.Run("nama kosong -> required", func(t *testing.T) {
		t.Parallel()
		err := validator.ValidateStruct(lengthDTO{})
		appErr := errorx.As(err)
		require.NotNil(t, appErr)
		details := appErr.Details
		assert.Equal(t, "Field 'name' wajib diisi", details["name"].(string))
	})

	t.Run("nama terlalu pendek -> min", func(t *testing.T) {
		t.Parallel()
		err := validator.ValidateStruct(lengthDTO{Name: "a", Pin: "123456"})
		appErr := errorx.As(err)
		require.NotNil(t, appErr)
		details := appErr.Details
		assert.Equal(t, "Field 'name' minimal 2 karakter/nilai", details["name"].(string))
	})

	t.Run("pin bukan 6 karakter -> len", func(t *testing.T) {
		t.Parallel()
		err := validator.ValidateStruct(lengthDTO{Name: "Andi", Pin: "12"})
		appErr := errorx.As(err)
		require.NotNil(t, appErr)
		details := appErr.Details
		assert.Equal(t, "Field 'pin' harus tepat 6 karakter", details["pin"].(string))
	})

	t.Run("valid -> NoError", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validator.ValidateStruct(lengthDTO{Name: "Andi", Pin: "123456"}))
	})
}

// TestValidator_CustomInstance memverifikasi bahwa setiap instance dapat
// dipakai terpisah dan tag name func tetap terdaftar.
func TestValidator_CustomInstance(t *testing.T) {
	t.Parallel()

	cv := validator.New()
	assert.NoError(t, cv.ValidateStruct(sampleDTO{
		Email:    "test@wesign.id",
		Password: "password123",
		Role:     "user",
		DocID:    "550e8400-e29b-41d4-a716-446655440000",
	}))

	err := cv.ValidateStruct(sampleDTO{Email: "bad"})
	appErr := errorx.As(err)
	require.NotNil(t, appErr)
	details := appErr.Details
	assert.Contains(t, details, "password", "struct kosong mencati seluruh field wajib")
}

// TestFormatFieldError_AllTags memuji pemetaan setiap tag validasi ke pesan
// Bahasa Indonesia, termasuk cabang default (tag yang belum dipetakan).
func TestFormatFieldError_AllTags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		field    string
		tag      string
		param    string
		expected string
	}{
		{"email", "required", "", "Field 'email' wajib diisi"},
		{"email", "email", "", "Field 'email' harus berupa alamat email yang valid"},
		{"password", "min", "8", "Field 'password' minimal 8 karakter/nilai"},
		{"password", "max", "32", "Field 'password' maksimal 32 karakter/nilai"},
		{"pin", "len", "6", "Field 'pin' harus tepat 6 karakter"},
		{"count", "gt", "0", "Field 'count' harus lebih besar dari 0"},
		{"count", "gte", "1", "Field 'count' harus lebih besar atau sama dengan 1"},
		{"count", "lt", "10", "Field 'count' harus lebih kecil dari 10"},
		{"count", "lte", "10", "Field 'count' harus lebih kecil atau sama dengan 10"},
		{"code", "numeric", "", "Field 'code' harus berupa angka"},
		{"username", "alphanum", "", "Field 'username' hanya boleh berisi huruf dan angka"},
		{"doc_id", "uuid", "", "Field 'doc_id' harus berupa format UUID yang valid"},
		{"role", "oneof", "admin user", "Field 'role' harus salah satu dari: admin user"},
		{"site", "url", "", "Field 'site' harus berupa URL yang valid"},
		{"unknown", "tag_manian", "", "Field 'unknown' tidak memenuhi aturan validasi 'tag_manian'"},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, validator.FormatFieldError(tc.field, tc.tag, tc.param))
		})
	}
}
