package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wesign/wesign-backend/internal/shared/errorx"
)

func TestNewSuccess(t *testing.T) {
	resp := NewSuccess(map[string]string{"message": "hello"})
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Meta)

	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"data"`)
	assert.Contains(t, string(b), `"hello"`)
	assert.NotContains(t, string(b), `"meta"`)
}

func TestNewSuccessWithMeta(t *testing.T) {
	resp := NewSuccessWithMeta([]string{"a", "b"}, map[string]int{"total": 2})
	assert.NotNil(t, resp.Data)
	assert.NotNil(t, resp.Meta)

	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"meta"`)
	assert.Contains(t, string(b), `"total"`)
}

func TestPaginated(t *testing.T) {
	items := []map[string]string{{"id": "1"}, {"id": "2"}}
	resp := Paginated(items, 10, "cursor_abc", true)

	b, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(b, &parsed)
	require.NoError(t, err)

	meta, ok := parsed["meta"].(map[string]any)
	require.True(t, ok, "meta should be a map")

	pagination, ok := meta["pagination"].(map[string]any)
	require.True(t, ok, "pagination should be a map")

	assert.Equal(t, float64(10), pagination["limit"])
	assert.Equal(t, "cursor_abc", pagination["next_cursor"])
	assert.Equal(t, true, pagination["has_more"])
}

func TestPaginated_NoNextCursor(t *testing.T) {
	resp := Paginated([]string{}, 20, "", false)

	b, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(b, &parsed)
	require.NoError(t, err)

	meta := parsed["meta"].(map[string]any)
	pagination := meta["pagination"].(map[string]any)

	assert.Equal(t, false, pagination["has_more"])
	_, hasCursor := pagination["next_cursor"]
	assert.False(t, hasCursor, "empty next_cursor should be omitted")
}

func TestNewError(t *testing.T) {
	resp := NewError("VALIDATION_ERROR", "Input tidak valid", "req-123", nil)

	b, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(b, &parsed)
	require.NoError(t, err)

	errObj := parsed["error"].(map[string]any)
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])
	assert.Equal(t, "Input tidak valid", errObj["message"])
	assert.Equal(t, "req-123", errObj["request_id"])
	_, hasDetails := errObj["details"]
	assert.False(t, hasDetails, "nil details should be omitted")
}

func TestFromAppError_NilError(t *testing.T) {
	resp := FromAppError(nil, "req-nil")
	assert.Equal(t, errorx.CodeInternal, resp.Error.Code)
	assert.Equal(t, "Terjadi kesalahan internal", resp.Error.Message)
	assert.Equal(t, "req-nil", resp.Error.RequestID)
}

func TestFromAppError_SanitizesSensitiveDetails(t *testing.T) {
	appErr := errorx.New(
		errorx.CodeInternal,
		"Server error",
	).WithDetails(map[string]any{
		"safe_key":          "visible",
		"internal_debug":    "hidden",
		"stack_trace":       "hidden too",
		"raw_error":         "also hidden",
		"sql_error":         "hidden",
		"db_error_info":     "hidden",
		"connection_string": "hidden",
	})

	resp := FromAppError(appErr, "req-sanitize")

	assert.Equal(t, errorx.CodeInternal, resp.Error.Code)
	assert.Equal(t, "req-sanitize", resp.Error.RequestID)

	details, ok := resp.Error.Details.(map[string]any)
	require.True(t, ok, "details should be a map after sanitization")
	assert.Equal(t, "visible", details["safe_key"])
	assert.NotContains(t, details, "internal_debug")
	assert.NotContains(t, details, "stack_trace")
	assert.NotContains(t, details, "raw_error")
	assert.NotContains(t, details, "sql_error")
	assert.NotContains(t, details, "db_error_info")
	assert.NotContains(t, details, "connection_string")
}

func TestFromAppError_AllSensitiveReturnsNilDetails(t *testing.T) {
	appErr := errorx.New(
		errorx.CodeInternal,
		"err",
	).WithDetails(map[string]any{
		"internal_key": "val",
		"stack_trace":  "trace",
	})

	resp := FromAppError(appErr, "req-all-sensitive")
	assert.Nil(t, resp.Error.Details, "all-sensitive details should become nil")
}

func TestSanitizeDetails_NestedMap(t *testing.T) {
	input := map[string]any{
		"safe": "ok",
		"nested": map[string]any{
			"deep_safe":      "visible",
			"internal_stuff": "hidden",
		},
	}

	result := SanitizeDetails(input)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", m["safe"])

	nested := m["nested"].(map[string]any)
	assert.Equal(t, "visible", nested["deep_safe"])
	assert.NotContains(t, nested, "internal_stuff")
}

func TestSanitizeDetails_Slice(t *testing.T) {
	input := []any{
		map[string]any{"safe": "ok", "raw_error": "hidden"},
		"plain string",
	}

	result := SanitizeDetails(input)
	s, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, s, 2)

	first := s[0].(map[string]any)
	assert.Equal(t, "ok", first["safe"])
	assert.NotContains(t, first, "raw_error")
	assert.Equal(t, "plain string", s[1])
}

func TestSanitizeDetails_PrimitivePassthrough(t *testing.T) {
	assert.Equal(t, "hello", SanitizeDetails("hello"))
	assert.Equal(t, 42, SanitizeDetails(42))
	assert.Nil(t, SanitizeDetails(nil))
}

func TestIsSensitiveDetailKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"internal_debug", true},
		{"INTERNAL_ERROR", true},
		{"stack_trace", true},
		{"StackTrace", true},
		{"raw_error", true},
		{"sql_error_detail", true},
		{"db_error", true},
		{"debug_info", true},
		{"connection_string", true},
		{"dsn", true},
		{"user_dsn_name", true},
		{"safe_key", false},
		{"message", false},
		{"field_name", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.sensitive, IsSensitiveDetailKey(tt.key))
		})
	}
}

func TestResponse_JSONRoundTrip(t *testing.T) {
	original := NewSuccessWithMeta(
		map[string]string{"id": "doc-123"},
		map[string]any{"version": 1},
	)

	b, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Response
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.Data)
	assert.NotNil(t, decoded.Meta)
}

func TestErrorResponse_JSONRoundTrip(t *testing.T) {
	original := NewError("AUTH_TOKEN_EXPIRED", "Token kedaluwarsa", "req-456", map[string]string{"hint": "refresh"})

	b, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ErrorResponse
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "AUTH_TOKEN_EXPIRED", decoded.Error.Code)
	assert.Equal(t, "Token kedaluwarsa", decoded.Error.Message)
	assert.Equal(t, "req-456", decoded.Error.RequestID)
}
