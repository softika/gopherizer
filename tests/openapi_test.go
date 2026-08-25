package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
)

// The document is generated from the registered operations, so it cannot drift
// from the routes the server actually serves.
func (s *E2ETestSuite) TestGeneratedOpenApiDocument() {
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Require().Equal(http.StatusOK, w.Code)

	var doc struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
		Comps   struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &doc))

	s.NotEmpty(doc.OpenAPI)

	// Every registered route must be documented.
	for path, methods := range map[string][]string{
		"/health":              {"get"},
		"/health/live":         {"get"},
		"/health/ready":        {"get"},
		"/api/v1/profile":      {"post", "put"},
		"/api/v1/profile/{id}": {"get", "delete"},
	} {
		ops, ok := doc.Paths[path]
		s.Require().Truef(ok, "path %s missing from the generated document", path)
		for _, m := range methods {
			s.Containsf(ops, m, "%s %s missing from the generated document", m, path)
		}
	}

	// Schemas are package-qualified, so same-named types do not collide.
	s.Contains(doc.Comps.Schemas, "ProfileResponse")
	s.Contains(doc.Comps.Schemas, "HealthResponse")

	// Field names come from the Go struct tags, so the camelCase/PascalCase
	// mismatch the hand-written spec had cannot recur.
	body := w.Body.String()
	s.Contains(body, "firstName")
	s.NotContains(body, `"FirstName"`)
}

func (s *E2ETestSuite) TestDocsUiIsServed() {
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	body := w.Body.String()
	s.NotEmpty(body)

	// huma defaults to Stoplight Elements, which renders a "powered by
	// Stoplight" credit. Assert the component is not loaded at all, which is
	// the only part observable server-side.
	s.NotContains(strings.ToLower(body), "stoplight",
		"the docs page must not load Stoplight Elements")

	// The renderer must still point at the generated document, in whichever
	// serialization it prefers.
	s.Contains(body, "/openapi.")

	// Third-party assets must carry a subresource integrity hash.
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "unpkg.com") {
			s.Containsf(line, "integrity=", "CDN asset loaded without SRI: %s", line)
		}
	}
}

// huma's default config installs a schema-link transformer that stamps a
// "$schema" field into every response body and a Link header alongside it.
// Both embed the server's own host, which leaks deployment detail into
// payloads, so the transformer is disabled.
func (s *E2ETestSuite) TestResponsesCarryNoSchemaLink() {
	body := bytes.NewReader([]byte(`{"firstName":"Grace","lastName":"Hopper"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profile", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Require().Equal(http.StatusCreated, w.Code)

	s.NotContains(w.Body.String(), "$schema", "response body must not carry a $schema field")
	s.NotContains(w.Body.String(), "localhost", "response body must not embed the server host")
	s.Empty(w.Header().Get("Link"), "response must not carry a schema Link header")

	// The document itself must still be served and complete.
	req = httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "ProfileResponse")
}
