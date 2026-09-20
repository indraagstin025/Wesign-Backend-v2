// Package validator membungkus github.com/go-playground/validator/v10 dan
// memetakan seluruh hasil validasi ke satu-satunya amplop error TDD §5.1:
//
//   - Struct tidak valid (field errors)      → 422 UNPROCESSABLE_ENTITY
//     dengan details: { "<field_json_tag>": "<pesan Bahasa Indonesia>" }
//   - Input bukan struct / format rusak      → 400 VALIDATION_ERROR
//
// Pesan dihasilkan berbasis kunci tag validasi (required, email, min, max,
// uuid, oneof, ...) sehingga siap untuk i18n (`Accept-Language: id|en`,
// TDD §5.1); penerjemahan penuh dicat di roadmap Fase 1 O-3.
//
// Nama field yang muncul pada key details mengikuti JSON tag (`json:"..."`),
// bukan nama field Go — menjaga konsistensi dengan OpenAPI wesignv2.yaml
// (mis. "doc_id" bukan "DocID").
package validator

import (
	"fmt"
	"reflect"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/wesign/wesign-backend/internal/shared/errorx"
)

// CustomValidator membungkus go-playground validator v10.
type CustomValidator struct {
	validator *govalidator.Validate
}

// New membuat CustomValidator dengan terdaftaran TagNameFunc sehingga nama
// field pada error key mengikuti JSON tag OpenAPI.
func New() *CustomValidator {
	v := govalidator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	return &CustomValidator{validator: v}
}

// defaultValidator adalah instance singleton yang aman dipakai bersama thread,
// karena govalidator.Validate bersifat bez-state setelah konfigurasi.
var defaultValidator = New()

// ValidateStruct memvalidasi struct menggunakan validator default.
func ValidateStruct(s any) error {
	return defaultValidator.ValidateStruct(s)
}

// ValidateStruct memvalidasi struct dan mengembalikan *errorx.AppError yang
// siap diserialisasi oleh pkg/response apabila validasi gagal.
func (cv *CustomValidator) ValidateStruct(s any) error {
	if err := cv.validator.Struct(s); err != nil {
		fieldErrors, ok := err.(govalidator.ValidationErrors)
		if !ok {
			// TDD §5.1: 400 VALIDATION_ERROR — request body tidak valid.
			return errorx.Validation("Format payload tidak valid")
		}

		details := make(map[string]any, len(fieldErrors))
		for _, f := range fieldErrors {
			details[f.Field()] = FormatFieldError(f.Field(), f.Tag(), f.Param())
		}
		return errorx.Unprocessable("Validasi input gagal").WithDetails(details)
	}

	return nil
}

// FormatFieldError memetakan tag validasi ke pesan Bahasa Indonesia yang
// komunikatif. Fungsi murni (bebas dari govalidator.FieldError) supaya bisa
// dietest secara langsung untuk seluruh tag.
func FormatFieldError(fieldName, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("Field '%s' wajib diisi", fieldName)
	case "email":
		return fmt.Sprintf("Field '%s' harus berupa alamat email yang valid", fieldName)
	case "min":
		return fmt.Sprintf("Field '%s' minimal %s karakter/nilai", fieldName, param)
	case "max":
		return fmt.Sprintf("Field '%s' maksimal %s karakter/nilai", fieldName, param)
	case "len":
		return fmt.Sprintf("Field '%s' harus tepat %s karakter", fieldName, param)
	case "gt":
		return fmt.Sprintf("Field '%s' harus lebih besar dari %s", fieldName, param)
	case "gte":
		return fmt.Sprintf("Field '%s' harus lebih besar atau sama dengan %s", fieldName, param)
	case "lt":
		return fmt.Sprintf("Field '%s' harus lebih kecil dari %s", fieldName, param)
	case "lte":
		return fmt.Sprintf("Field '%s' harus lebih kecil atau sama dengan %s", fieldName, param)
	case "numeric":
		return fmt.Sprintf("Field '%s' harus berupa angka", fieldName)
	case "alphanum":
		return fmt.Sprintf("Field '%s' hanya boleh berisi huruf dan angka", fieldName)
	case "uuid":
		return fmt.Sprintf("Field '%s' harus berupa format UUID yang valid", fieldName)
	case "oneof":
		return fmt.Sprintf("Field '%s' harus salah satu dari: %s", fieldName, param)
	case "url":
		return fmt.Sprintf("Field '%s' harus berupa URL yang valid", fieldName)
	default:
		return fmt.Sprintf("Field '%s' tidak memenuhi aturan validasi '%s'", fieldName, tag)
	}
}
