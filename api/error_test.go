package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/pkg/errorx"
)

// dbErr stands in for a raw driver failure: the kind of detail that must never
// reach an API client.
const dbErrText = `ERROR: relation "profiles" does not exist (SQLSTATE 42P01) host=10.0.3.7`

func TestNewServiceErrorMapsStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errType  errorx.ErrorType
		wantCode int
	}{
		{"invalid input", errorx.ErrInvalidInput, http.StatusBadRequest},
		{"unauthorized", errorx.ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", errorx.ErrForbidden, http.StatusForbidden},
		{"not found", errorx.ErrNotFound, http.StatusNotFound},
		{"internal", errorx.ErrInternal, http.StatusInternalServerError},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svcErr := errorx.NewError(errors.New(dbErrText), tt.errType)
			assert.Equal(t, tt.wantCode, newServiceError(svcErr).Code)
		})
	}
}

func TestNewServiceErrorUntypedIsInternal(t *testing.T) {
	t.Parallel()

	apiErr := newServiceError(errors.New(dbErrText))
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

// TestNewServiceErrorDoesNotLeakInternals is the guard for the actual defect:
// the serialized response must carry no driver or infrastructure detail.
func TestNewServiceErrorDoesNotLeakInternals(t *testing.T) {
	t.Parallel()

	for _, errType := range []errorx.ErrorType{
		errorx.ErrInternal,
		errorx.ErrInvalidInput,
		errorx.ErrForbidden,
		errorx.ErrNotFound,
		errorx.ErrUnauthorized,
	} {
		svcErr := errorx.NewError(
			fmt.Errorf("failed to get profile by id: %w", errors.New(dbErrText)),
			errType,
		)

		apiErr := newServiceError(svcErr)
		body := apiErr.Error()

		for _, leak := range []string{"relation", "SQLSTATE", "42P01", "host=", "profiles"} {
			assert.NotContainsf(t, body, leak,
				"response body leaked %q for error type %v: %s", leak, errType, body)
		}

		assert.NotEmpty(t, apiErr.Message, "a safe message must still be present")

		// The full detail must survive for server-side logging.
		require.Error(t, apiErr.Cause)
		assert.Contains(t, apiErr.Cause.Error(), dbErrText)
	}
}

func TestErrorSerializesWithoutCause(t *testing.T) {
	t.Parallel()

	apiErr := newServiceError(errorx.NewError(errors.New(dbErrText), errorx.ErrNotFound))

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(apiErr.Error()), &payload))

	_, hasCause := payload["cause"]
	assert.False(t, hasCause, "cause must never be serialized")
	assert.EqualValues(t, http.StatusNotFound, payload["code"])
}

// TestErrorWithNilCause guards the fallback branch against a nil dereference.
func TestErrorWithNilCause(t *testing.T) {
	t.Parallel()

	apiErr := Error{Code: http.StatusInternalServerError, Message: "boom"}
	assert.NotPanics(t, func() {
		assert.True(t, strings.Contains(apiErr.Error(), "boom"))
	})
}
