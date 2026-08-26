package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/softika/gopherizer/pkg/errorx"
)

// classify tags a driver failure with a domain error type.
//
// The repository is the only layer that knows pgx, so it is the only place that
// can tell a missing row apart from a genuine failure. Everything above works
// with errorx types instead of driver sentinels.
func classify(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorx.NewError(err, errorx.ErrNotFound)
	}

	// A request that ran out of time, or whose caller went away, is not a
	// server fault. Reporting 500 would tell a caller the service is broken
	// when it was merely too slow for this request -- and would put a genuine
	// fault and an ordinary timeout in the same bucket on every dashboard.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errorx.NewError(err, errorx.ErrUnavailable)
	}

	return errorx.NewError(err, errorx.ErrInternal)
}

// notFound reports a row that the statement did not match.
func notFound(entity, id string) error {
	return errorx.NewError(
		fmt.Errorf("%s %q not found", entity, id),
		errorx.ErrNotFound,
	)
}
