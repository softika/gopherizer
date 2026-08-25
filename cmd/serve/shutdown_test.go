package serve

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// recorder captures the order in which resources are released.
type recorder struct {
	seq []string
}

type fakeHTTPServer struct {
	rec *recorder
	err error
}

func (f *fakeHTTPServer) Shutdown(context.Context) error {
	f.rec.seq = append(f.rec.seq, "server")
	return f.err
}

type fakeDB struct {
	rec *recorder
	err error
}

func (f *fakeDB) Close() error {
	f.rec.seq = append(f.rec.seq, "db")
	return f.err
}

func TestShutdown(t *testing.T) {
	srvErr := errors.New("server shutdown failed")
	dbErr := errors.New("db close failed")

	cases := []struct {
		name     string
		srvErr   error
		dbErr    error
		wantErrs []error
		wantSeq  []string
	}{
		{
			name:    "releases both in order",
			wantSeq: []string{"server", "db"},
		},
		{
			name:     "closes the pool even when server shutdown fails",
			srvErr:   srvErr,
			wantErrs: []error{srvErr},
			wantSeq:  []string{"server", "db"},
		},
		{
			name:     "reports a pool close failure",
			dbErr:    dbErr,
			wantErrs: []error{dbErr},
			wantSeq:  []string{"server", "db"},
		},
		{
			name:     "reports both failures",
			srvErr:   srvErr,
			dbErr:    dbErr,
			wantErrs: []error{srvErr, dbErr},
			wantSeq:  []string{"server", "db"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{}
			srv := &fakeHTTPServer{rec: rec, err: tc.srvErr}
			db := &fakeDB{rec: rec, err: tc.dbErr}

			err := shutdown(context.Background(), srv, db)

			for _, want := range tc.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("expected error to wrap %v, got %v", want, err)
				}
			}
			if len(tc.wantErrs) == 0 && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(rec.seq, tc.wantSeq) {
				t.Errorf("expected release order %v, got %v", tc.wantSeq, rec.seq)
			}
		})
	}
}
