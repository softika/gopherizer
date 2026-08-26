package repositories

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/pkg/errorx"
)

// TestClassifyTimeouts keeps a request that ran out of time from being reported
// as a server fault. A 500 tells a caller the service is broken; a timeout
// means it was too slow for this request, which is a different thing to act on.
func TestClassifyTimeouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorx.ErrorType
	}{
		{"no rows is not found", pgx.ErrNoRows, errorx.ErrNotFound},
		{"wrapped no rows is not found", fmt.Errorf("query: %w", pgx.ErrNoRows), errorx.ErrNotFound},
		{"deadline exceeded is unavailable", context.DeadlineExceeded, errorx.ErrUnavailable},
		{"wrapped deadline is unavailable", fmt.Errorf("query: %w", context.DeadlineExceeded), errorx.ErrUnavailable},
		{"cancellation is unavailable", context.Canceled, errorx.ErrUnavailable},
		{"anything else is internal", errors.New("boom"), errorx.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, errorx.TypeOf(classify(tt.err)))
		})
	}
}
