package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// seededProfileId is John Doe, from tests/testdata. Read-only here: the
// mutating cases below create their own records so they cannot disturb the
// tests that share this one.
const seededProfileId = "0dd35f9a-0d20-41f1-80c2-d7993e313fb4"

// bearer mints a token the secured router will accept, carrying the given
// scopes and roles.
func (s *E2ETestSuite) bearer(scopes string, roles ...string) string {
	claims := map[string]any{"sub": "e2e-user", "scope": scopes}
	if len(roles) > 0 {
		claims["roles"] = roles
	}

	token, err := s.oidc.Token(claims)
	s.Require().NoError(err)

	return "Bearer " + token
}

func (s *E2ETestSuite) securedCall(method, path, authorization string, body []byte) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	w := httptest.NewRecorder()
	s.securedRouter.ServeHTTP(w, req)

	return w
}

// An orchestrator carries no token, so the probes have to answer for one.
func (s *E2ETestSuite) TestSecuredRouterLeavesProbesAndDocsOpen() {
	for _, path := range []string{"/", "/health/live", "/health/ready", "/health", "/metrics", "/docs", "/openapi.json"} {
		// given an unauthenticated caller
		// when the operational endpoint is requested
		w := s.securedCall(http.MethodGet, path, "", nil)

		// then it answers normally
		s.Equalf(http.StatusOK, w.Code, "%s must stay reachable without a token", path)
	}
}

func (s *E2ETestSuite) TestSecuredRouterRejectsAnonymousProfileAccess() {
	// given a profile that exists, and no credential
	// when it is requested
	w := s.securedCall(http.MethodGet, "/api/v1/profile/"+seededProfileId, "", nil)

	// then the caller is challenged rather than served
	s.Equal(http.StatusUnauthorized, w.Code)
	s.Equal(`Bearer error="invalid_token"`, w.Header().Get("WWW-Authenticate"))
	s.NotContains(w.Body.String(), "firstName", "no part of the record may reach an unauthenticated caller")
}

func (s *E2ETestSuite) TestSecuredRouterAcceptsASufficientToken() {
	// given a token carrying the read scope
	// when the profile is requested
	w := s.securedCall(http.MethodGet, "/api/v1/profile/"+seededProfileId, s.bearer("profile:read"), nil)

	// then it is served
	s.Require().Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "firstName")
}

// A verified caller who lacks the scope is refused, and told so distinctly from
// one who presented nothing at all.
func (s *E2ETestSuite) TestSecuredRouterRefusesAnInsufficientToken() {
	w := s.securedCall(http.MethodGet, "/api/v1/profile/"+seededProfileId, s.bearer("something:else"), nil)

	s.Equal(http.StatusForbidden, w.Code)
	s.Empty(w.Header().Get("WWW-Authenticate"))
}

func (s *E2ETestSuite) TestSecuredRouterEnforcesWriteScopeOnCreate() {
	body, err := json.Marshal(map[string]string{"firstName": "Grace", "lastName": "Hopper"})
	s.Require().NoError(err)

	// given a token that may only read
	readOnly := s.securedCall(http.MethodPost, "/api/v1/profile", s.bearer("profile:read"), body)
	s.Equal(http.StatusForbidden, readOnly.Code, "the read scope must not permit a write")

	// given a token that may write
	writer := s.securedCall(http.MethodPost, "/api/v1/profile", s.bearer("profile:write"), body)
	s.Equal(http.StatusCreated, writer.Code)
}

// Delete requires a role as well as a scope, which is the part an OpenAPI
// security requirement cannot express and the guard therefore has to carry.
func (s *E2ETestSuite) TestSecuredRouterEnforcesARoleThatTheDocumentCannotExpress() {
	body, err := json.Marshal(map[string]string{"firstName": "Ada", "lastName": "Lovelace"})
	s.Require().NoError(err)

	created := s.securedCall(http.MethodPost, "/api/v1/profile", s.bearer("profile:write"), body)
	s.Require().Equal(http.StatusCreated, created.Code)

	var profile struct {
		Id string `json:"id"`
	}
	s.Require().NoError(json.Unmarshal(created.Body.Bytes(), &profile))

	// given the write scope but no role
	withoutRole := s.securedCall(http.MethodDelete, "/api/v1/profile/"+profile.Id, s.bearer("profile:write"), nil)
	s.Equal(http.StatusForbidden, withoutRole.Code, "the scope alone must not permit a delete")

	// given the write scope and the role
	withRole := s.securedCall(http.MethodDelete, "/api/v1/profile/"+profile.Id, s.bearer("profile:write", "admin"), nil)
	s.Equal(http.StatusNoContent, withRole.Code)
}

// The document must advertise the scheme the service is actually enforcing.
func (s *E2ETestSuite) TestSecuredRouterDocumentsItsSecurity() {
	w := s.securedCall(http.MethodGet, "/openapi.json", "", nil)
	s.Require().Equal(http.StatusOK, w.Code)

	var doc struct {
		Paths map[string]map[string]struct {
			Security []map[string][]string `json:"security"`
		} `json:"paths"`
		Comps struct {
			SecuritySchemes map[string]struct {
				Type   string `json:"type"`
				Scheme string `json:"scheme"`
			} `json:"securitySchemes"`
		} `json:"components"`
	}
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &doc))

	scheme, found := doc.Comps.SecuritySchemes["bearerAuth"]
	s.Require().True(found, "an enforcing service must declare its scheme")
	s.Equal("http", scheme.Type)
	s.Equal("bearer", scheme.Scheme)

	// The read operation documents the scope it enforces.
	s.Equal(
		[]map[string][]string{{"bearerAuth": {"profile:read"}}},
		doc.Paths["/api/v1/profile/{id}"]["get"].Security,
	)

	// Delete additionally requires a role, which OpenAPI cannot express -- so
	// it documents the scope only, understating rather than overstating.
	s.Equal(
		[]map[string][]string{{"bearerAuth": {"profile:write"}}},
		doc.Paths["/api/v1/profile/{id}"]["delete"].Security,
		"a role requirement must not leak into the document as a scope",
	)

	// The probes stay undeclared, matching the fact that they are unguarded.
	s.Empty(doc.Paths["/health/live"]["get"].Security)
}

// The provider being down must reach the caller as 503, through the whole
// stack rather than through a hand-built error.
//
// This is the shape the stub could not prove: go-oidc flattens the transport
// failure into a string with %v, so an error constructed in a test preserves a
// chain the real library destroys. Only a genuinely broken provider exercises
// the classification that matters.
func (s *E2ETestSuite) TestSecuredRouterReportsAProviderOutageAsUnavailable() {
	// given a key id the verifier has not cached, so the key set must be
	// fetched rather than read from memory
	token, err := s.oidc.RSAToken(map[string]any{"sub": "e2e-user", "scope": "profile:read"})
	s.Require().NoError(err)

	// and a provider whose key endpoint is failing
	s.oidc.FailJWKS(http.StatusServiceUnavailable)
	defer s.oidc.FailJWKS(0)

	// when a request is made with an otherwise valid token
	w := s.securedCall(http.MethodGet, "/api/v1/profile/"+seededProfileId, "Bearer "+token, nil)

	// then the caller learns the service could not check, not that they were refused
	s.Equal(http.StatusServiceUnavailable, w.Code,
		"an outage at the provider must not be reported as a bad credential")
	s.Empty(w.Header().Get("WWW-Authenticate"),
		"challenging for another credential is the wrong advice when none would have worked")
}
