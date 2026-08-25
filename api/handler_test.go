package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/pkg/errorx"
)

// steps records which stages of the pipeline ran, and in what order.
type steps struct {
	seq []string
}

func (s *steps) decoder(in sample, err error) Decoder[sample] {
	return func(*http.Request) (sample, error) {
		s.seq = append(s.seq, "decode")
		return in, err
	}
}

func (s *steps) service(out string, err error) ServiceFunc[sample, string] {
	return func(context.Context, sample) (string, error) {
		s.seq = append(s.seq, "service")
		return out, err
	}
}

func (s *steps) encoder(err error) Encoder[string] {
	return func(http.ResponseWriter, string) error {
		s.seq = append(s.seq, "encode")
		return err
	}
}

type fakeValidator struct {
	steps *steps
	err   error
}

func (f fakeValidator) StructCtx(context.Context, any) error {
	f.steps.seq = append(f.steps.seq, "validate")
	return f.err
}

func TestHandlerRunsPipelineInOrder(t *testing.T) {
	t.Parallel()

	s := &steps{}
	h := NewHandler(
		s.decoder(sample{Name: "John"}, nil),
		s.encoder(nil),
		s.service("ok", nil),
		fakeValidator{steps: s},
	)

	err := h.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	require.NoError(t, err)
	assert.Equal(t, []string{"decode", "validate", "service", "encode"}, s.seq)
}

// Each stage must stop the pipeline, so a later stage never runs on input an
// earlier one rejected.
func TestHandlerStopsAtFirstFailure(t *testing.T) {
	t.Parallel()

	decodeErr := errors.New("bad body")
	validateErr := errors.New("missing field")
	serviceErr := errorx.NewError(errors.New("no rows"), errorx.ErrNotFound)

	tests := []struct {
		name     string
		decErr   error
		vldErr   error
		svcErr   error
		wantSeq  []string
		wantCode int
	}{
		{
			name:     "decode failure is a bad request",
			decErr:   decodeErr,
			wantSeq:  []string{"decode"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "validation failure is a bad request",
			vldErr:   validateErr,
			wantSeq:  []string{"decode", "validate"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service failure keeps its mapped status",
			svcErr:   serviceErr,
			wantSeq:  []string{"decode", "validate", "service"},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &steps{}
			h := NewHandler(
				s.decoder(sample{}, tt.decErr),
				s.encoder(nil),
				s.service("", tt.svcErr),
				fakeValidator{steps: s, err: tt.vldErr},
			)

			err := h.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

			require.Error(t, err)
			assert.Equal(t, tt.wantSeq, s.seq, "the pipeline must stop at the failing stage")

			var apiErr Error
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestHandlerSkipsValidationWhenUnset(t *testing.T) {
	t.Parallel()

	s := &steps{}
	h := NewHandler(s.decoder(sample{}, nil), s.encoder(nil), s.service("ok", nil), nil)

	require.NoError(t, h.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil)))
	assert.Equal(t, []string{"decode", "service", "encode"}, s.seq)
}

func TestHandlerPropagatesEncodeFailure(t *testing.T) {
	t.Parallel()

	s := &steps{}
	encodeErr := errors.New("broken pipe")
	h := NewHandler(s.decoder(sample{}, nil), s.encoder(encodeErr), s.service("ok", nil), nil)

	err := h.Handle(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	assert.ErrorIs(t, err, encodeErr)
}
