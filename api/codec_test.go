package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name string `json:"name"`
}

type pathInput struct {
	Id string
}

func TestJSONBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		want    sample
		wantErr bool
	}{
		{name: "decodes an object", body: `{"name":"John"}`, want: sample{Name: "John"}},
		{name: "unknown fields are ignored", body: `{"name":"John","extra":1}`, want: sample{Name: "John"}},
		{name: "rejects malformed json", body: `{`, wantErr: true},
		{name: "rejects an empty body", body: ``, wantErr: true},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			got, err := JSONBody[sample](r)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPathParam(t *testing.T) {
	t.Parallel()

	decode := PathParam("id", func(v string) pathInput { return pathInput{Id: v} })

	t.Run("builds the input from the parameter", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetPathValue("id", "abc")

		got, err := decode(r)
		require.NoError(t, err)
		assert.Equal(t, pathInput{Id: "abc"}, got)
	})

	t.Run("reports a missing parameter", func(t *testing.T) {
		t.Parallel()

		_, err := decode(httptest.NewRequest(http.MethodGet, "/", nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id")
	})
}

func TestStatic(t *testing.T) {
	t.Parallel()

	got, err := Static(sample{Name: "fixed"})(httptest.NewRequest(http.MethodGet, "/", nil))

	require.NoError(t, err)
	assert.Equal(t, sample{Name: "fixed"}, got)
}

func TestJSONEncoder(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	require.NoError(t, JSON[sample](http.StatusCreated)(w, sample{Name: "John"}))

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"name":"John"}`, w.Body.String())
}

// A 204 must carry no body: net/http discards one anyway, so writing it is a
// silent no-op that misleads whoever reads the code.
func TestNoContentWritesNoBody(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	require.NoError(t, NoContent(w, true))

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.Empty(t, w.Header().Get("Content-Type"))
}
