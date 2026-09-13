package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/authx"
)

// A syntactically valid id, so a request reaches the guard rather than being
// turned away by the path pattern before it.
const someProfileId = "00000000-0000-0000-0000-000000000001"

// authConfig mirrors the shipped defaults, built by hand rather than through
// config.New so these tests neither touch the process-wide viper instance nor
// need to run serially.
func authConfig(enabled bool) *config.Config {
	httpCfg := config.HTTPConfig{Port: "8080"}
	httpCfg.Metrics.Enabled = true
	httpCfg.Metrics.Path = "/metrics"

	return &config.Config{
		App:  config.AppConfig{Name: "gopherizer", Environment: "test", Version: "1.0.0"},
		Http: httpCfg,
		Oidc: config.OIDCConfig{
			Enabled:  enabled,
			Issuer:   "https://issuer.example",
			Audience: "gopherizer-api",
		},
	}
}

func authRouter(t *testing.T, id authx.Identity) *Router {
	t.Helper()

	return NewRouter(authConfig(true), stubDB{}, WithVerifier(stubVerifier{id: id}))
}

func call(t *testing.T, r *Router, method, path string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	for _, h := range headers {
		name, value, _ := strings.Cut(h, ": ")
		req.Header.Set(name, value)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

// Enabling OIDC without supplying a verifier must not produce a running service
// that serves every protected route to anyone. NewRouter returns no error, so
// the only exit that fails closed is a panic.
func TestNewRouterPanicsWhenOidcIsEnabledWithoutAVerifier(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t,
		"api: oidc is enabled but no verifier was supplied; pass api.WithVerifier",
		func() { NewRouter(authConfig(true), stubDB{}) },
	)
}

// Everything an operator, an orchestrator or a browser needs must stay
// reachable. These are served outside the registered operations -- by chi, or
// by huma's own document handlers -- which is why the guard is attached per
// operation rather than to the whole router.
func TestAuthLeavesOperationalEndpointsOpen(t *testing.T) {
	t.Parallel()

	r := authRouter(t, authx.Identity{Subject: "user-1"})

	for _, path := range []string{"/", "/health/live", "/health/ready", "/health", "/metrics", "/docs", "/openapi.json"} {
		p := path
		t.Run(p, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, http.StatusOK, call(t, r, http.MethodGet, p).Code,
				"%s must stay reachable without a token", p)
		})
	}
}

func TestAuthProtectsTheProfileRoutes(t *testing.T) {
	t.Parallel()

	r := authRouter(t, authx.Identity{Subject: "user-1"})

	res := call(t, r, http.MethodGet, "/api/v1/profile/"+someProfileId)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, `Bearer error="invalid_token"`, res.Header().Get("WWW-Authenticate"))
}

// The scheme must appear in the document exactly when the guard is enforcing.
// Either half without the other is a document that lies: one advertises a check
// nothing performs, the other hides a check every caller will hit.
func TestSecuritySchemeIsDocumentedIffEnforced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var r *Router
			if tt.enabled {
				r = authRouter(t, authx.Identity{Subject: "user-1"})
			} else {
				r = NewRouter(authConfig(false), stubDB{})
			}

			doc := openAPIDocument(t, r)

			schemes, found := doc.Components.SecuritySchemes[securitySchemeName]
			assert.Equal(t, tt.enabled, found, "the scheme must be present exactly when enforcement is on")

			if tt.enabled {
				assert.Equal(t, "http", schemes.Type)
				assert.Equal(t, "bearer", schemes.Scheme)
			}

			// And the same, seen from the operations rather than the components.
			secured := securedOperations(doc)
			if tt.enabled {
				assert.NotEmpty(t, secured, "enforced operations must declare their requirement")
				return
			}
			assert.Empty(t, secured, "an unauthenticated service must not advertise requirements")
		})
	}
}

// Derived from the generated document rather than from a list maintained here,
// so an endpoint added later is covered the day it is added rather than the day
// somebody remembers to add it below.
func TestEverySecuredOperationRejectsAnonymousCallers(t *testing.T) {
	t.Parallel()

	r := authRouter(t, authx.Identity{Subject: "user-1"})

	secured := securedOperations(openAPIDocument(t, r))
	require.NotEmpty(t, secured, "the document must declare at least one secured operation for this to prove anything")

	for _, op := range secured {
		t.Run(op.method+" "+op.path, func(t *testing.T) {
			t.Parallel()

			path := strings.ReplaceAll(op.path, "{id}", someProfileId)

			res := call(t, r, strings.ToUpper(op.method), path)

			assert.Equal(t, http.StatusUnauthorized, res.Code,
				"%s %s advertises a security requirement and must enforce it", op.method, op.path)
		})
	}
}

// Every profile operation must be guarded. Health is excluded deliberately --
// see registerHealth.
func TestEveryProfileOperationDeclaresSecurity(t *testing.T) {
	t.Parallel()

	r := authRouter(t, authx.Identity{Subject: "user-1"})
	doc := openAPIDocument(t, r)

	for path, methods := range doc.Paths {
		if !strings.HasPrefix(path, "/api/") {
			continue
		}

		for method, op := range methods {
			assert.NotEmptyf(t, op.Security,
				"%s %s is under /api and declares no security requirement", method, path)
		}
	}
}

type openAPIDoc struct {
	Paths map[string]map[string]struct {
		Security []map[string][]string `json:"security"`
	} `json:"paths"`
	Components struct {
		SecuritySchemes map[string]struct {
			Type   string `json:"type"`
			Scheme string `json:"scheme"`
		} `json:"securitySchemes"`
	} `json:"components"`
}

func openAPIDocument(t *testing.T, r *Router) openAPIDoc {
	t.Helper()

	res := call(t, r, http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, res.Code)

	var doc openAPIDoc
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &doc))

	return doc
}

type operationRef struct {
	method string
	path   string
}

func securedOperations(doc openAPIDoc) []operationRef {
	var out []operationRef

	for path, methods := range doc.Paths {
		for method, op := range methods {
			if len(op.Security) > 0 {
				out = append(out, operationRef{method: method, path: path})
			}
		}
	}

	return out
}
