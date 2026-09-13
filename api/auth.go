package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/authx"
	"github.com/softika/gopherizer/pkg/errorx"
	"github.com/softika/gopherizer/pkg/oidcx"
)

const (
	// securitySchemeName keys both components.securitySchemes and every
	// operation's security requirement. Stated once, because the two have to
	// agree for a client to resolve the reference.
	securitySchemeName = "bearerAuth"

	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

// guard enforces an operation's authorization requirement.
//
// Enforcement is attached per operation rather than through the API-wide
// middleware stack, and that is a correctness decision rather than a stylistic
// one. huma evaluates api.Middlewares() inside huma.Register and bakes the
// result into that operation's handler, so a stack modified after the
// operations were registered is silently ignored -- every operation would
// enforce nothing while its Security field still advertised the scheme. Per
// operation there is no ordering left to get wrong: see secured.
type guard struct {
	api      huma.API
	verifier oidcx.Verifier
	logger   *slog.Logger
}

func newGuard(api huma.API, verifier oidcx.Verifier, logger *slog.Logger) *guard {
	return &guard{api: api, verifier: verifier, logger: logger}
}

// securityScheme describes how a caller authenticates, for the generated
// document.
//
// Declared as http/bearer rather than openIdConnect: this service is a resource
// server with no login flow of its own, and the openIdConnect type would have
// the docs UI offer one it cannot complete. http/bearer renders an Authorize
// box that takes a pasted token, which is the interaction actually on offer.
//
// BearerFormat is left unset. The template does not require the token be a JWT
// anywhere a client could observe, and naming a format it does not enforce
// would be documentation drifting from behaviour.
func securityScheme(cfg config.OIDCConfig) *huma.SecurityScheme {
	description := "OIDC access token issued by " + cfg.Issuer +
		", presented as `Authorization: Bearer <token>`."

	return &huma.SecurityScheme{
		Type:        "http",
		Scheme:      "bearer",
		Description: description,
	}
}

// secured returns op with its authorization requirement both documented and
// enforced.
//
// The two are set together, from one argument, in one statement. That is the
// whole point: an operation cannot advertise a requirement it does not enforce,
// because there is no way to write the first without also writing the second.
//
// A nil guard means the service authenticates nothing, so the requirement is
// dropped rather than half-applied -- the generated document then describes an
// unauthenticated service honestly, instead of advertising a scheme that
// nothing checks. That is the only condition under which an operation passed
// through here is left open.
//
// In particular an empty requirement does not reopen it. A requirement is
// frequently an expression -- authx.AllOf over a term read from configuration,
// say -- and such an expression can evaluate to empty. Reading that as "no
// requirement was declared" would turn a rule nobody can satisfy into a public
// endpoint, inverting its meaning at exactly the moment it matters. Enforcing
// it instead refuses everyone, which is loud, immediate and safe. An operation
// that is meant to be open does not come through this function at all; see
// registerHealth.
func secured(op huma.Operation, req authx.Requirements, g *guard) huma.Operation {
	if g == nil {
		return op
	}

	security := req.OpenAPI(securitySchemeName)
	if security == nil {
		// Nothing is satisfiable, so nothing can be described as sufficient.
		// Documenting "a token is required" understates the refusal rather
		// than overstating what will be accepted, which is the direction this
		// projection is allowed to err in.
		security = []map[string][]string{{securitySchemeName: {}}}
	}

	op.Security = security
	op.Middlewares = append(op.Middlewares, g.middleware(req))
	op.Errors = append(op.Errors,
		http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable)

	return op
}

// middleware builds the check for one operation's requirement.
//
// It runs before the handler, and therefore before huma reads or validates the
// request body: an unauthenticated caller cannot make the service parse
// anything it sent.
func (g *guard) middleware(req authx.Requirements) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token, ok := bearerToken(ctx.Header(authorizationHeader))
		if !ok {
			g.deny(ctx, errorx.ErrUnauthorized, "no bearer token presented")
			return
		}

		id, err := g.verifier.Verify(ctx.Context(), token)
		if err != nil {
			// TypeOf keeps the verifier's own distinction: a provider that
			// cannot be reached answers 503, not 401, so an outage there does
			// not read as every caller suddenly holding bad credentials. An
			// untyped error stays internal, which fails closed rather than
			// being charitably read as a rejection.
			g.deny(ctx, errorx.TypeOf(err), "token rejected", "error", err)
			return
		}

		if !req.Satisfied(id) {
			g.deny(ctx, errorx.ErrForbidden, "requirement not satisfied",
				"subject", id.Subject,
				"required", req.String(),
			)

			return
		}

		next(huma.WithContext(ctx, authx.NewContext(ctx.Context(), id)))
	}
}

// deny answers the caller and records the decision.
//
// The response carries the same generic wording every other error of its type
// does, and never the reason: naming the failed check tells an attacker which
// part they already have right. The reason goes to the log, where the
// correlation middleware has already attached the request id that ties it back.
//
// The credential itself is never logged, at any level.
//
// One record per refusal, whatever the cause: extra is folded in here rather
// than logged separately, so a denial cannot appear as two unrelated lines that
// an operator has to join by timestamp.
func (g *guard) deny(ctx huma.Context, errType errorx.ErrorType, reason string, extra ...any) {
	status := statusError(errType)

	if errType == errorx.ErrUnauthorized {
		// RFC 7235 requires a challenge alongside a 401. The error code is
		// named because RFC 6750 defines it; no description is offered, for
		// the same reason the body carries none.
		ctx.SetHeader("WWW-Authenticate", `Bearer error="invalid_token"`)
	}

	attrs := append([]any{
		"operation", ctx.Operation().OperationID,
		"reason", reason,
		"status", status.GetStatus(),
	}, extra...)

	g.logger.WarnContext(ctx.Context(), "request denied", attrs...)

	// No errs argument: huma renders every one it is given into the
	// client-visible body, which is where an internal detail would leak.
	_ = huma.WriteErr(g.api, ctx, status.GetStatus(), status.Error())
}

// bearerToken extracts the credential from an Authorization header value.
//
// Anything that is not exactly one Bearer token with a non-empty value is
// reported as absent, so a malformed header is denied rather than being read as
// an anonymous request that some later check might wave through. The scheme is
// compared case-insensitively, as RFC 7235 requires.
func bearerToken(header string) (string, bool) {
	if len(header) < len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])

	return token, token != ""
}
