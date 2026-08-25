package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	s.NotEmpty(w.Body.String())
}
