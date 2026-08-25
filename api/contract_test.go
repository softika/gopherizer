package api

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/internal/profile"
)

// jsonFieldNames returns the wire names a type serializes to.
func jsonFieldNames(t *testing.T, v any) []string {
	t.Helper()

	rt := reflect.TypeOf(v)
	names := make([]string, 0, rt.NumField())

	for i := range rt.NumField() {
		f := rt.Field(i)

		tag, ok := f.Tag.Lookup("json")
		require.Truef(t, ok, "%s.%s has no json tag, so its wire name is implicit", rt.Name(), f.Name)

		name := strings.Split(tag, ",")[0]
		require.NotEmptyf(t, name, "%s.%s has an empty json tag", rt.Name(), f.Name)

		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func schemaPropertyNames(t *testing.T, doc *openapi3.T, schema string) []string {
	t.Helper()

	ref, ok := doc.Components.Schemas[schema]
	require.Truef(t, ok, "schema %q missing from the openapi document", schema)

	names := make([]string, 0, len(ref.Value.Properties))
	for name := range ref.Value.Properties {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// TestOpenApiMatchesGoTypes keeps the published contract honest.
//
// The document is what clients generate from; if it drifts from the structs the
// server actually marshals, every generated client is silently wrong.
func TestOpenApiMatchesGoTypes(t *testing.T) {
	t.Parallel()

	loader := &openapi3.Loader{Context: context.Background(), IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile("docs/v1/api.yaml")
	require.NoError(t, err)

	tests := []struct {
		schema string
		value  any
	}{
		{"ProfileResponse", profile.Response{}},
		{"CreateRequest", profile.CreateRequest{}},
		{"UpdateRequest", profile.UpdateRequest{}},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.schema, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				jsonFieldNames(t, tt.value),
				schemaPropertyNames(t, doc, tt.schema),
				"openapi schema %q does not match the Go type it documents", tt.schema,
			)
		})
	}
}
