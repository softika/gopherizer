package errorx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/pkg/errorx"
)

func TestTypeOf(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("db: no rows in result set")

	tests := []struct {
		name string
		err  error
		want errorx.ErrorType
	}{
		{
			name: "nil error defaults to internal",
			err:  nil,
			want: errorx.ErrInternal,
		},
		{
			name: "untyped error defaults to internal",
			err:  sentinel,
			want: errorx.ErrInternal,
		},
		{
			name: "typed error reports its type",
			err:  errorx.NewError(sentinel, errorx.ErrNotFound),
			want: errorx.ErrNotFound,
		},
		{
			name: "typed error found through a wrap chain",
			err:  fmt.Errorf("service failed: %w", errorx.NewError(sentinel, errorx.ErrForbidden)),
			want: errorx.ErrForbidden,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, errorx.TypeOf(tt.err))
		})
	}
}

// TestErrorUnwrap keeps errors.Is working through the domain error, so callers
// can still match on driver sentinels when they need to.
func TestErrorUnwrap(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("no rows in result set")
	err := errorx.NewError(fmt.Errorf("get profile: %w", sentinel), errorx.ErrNotFound)

	assert.True(t, errors.Is(err, sentinel), "expected errors.Is to reach the wrapped sentinel")
}
