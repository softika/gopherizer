// Package oidcx verifies OIDC bearer tokens and resolves them to an identity.
//
// It is the only place in the module that imports go-oidc, so the identity
// provider stays replaceable and authx stays dependency free. Nothing here
// knows about HTTP routing, huma or chi: it turns a credential into an
// authx.Identity, and the api package decides what that identity may do.
package oidcx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/authx"
)

const (
	// defaultDiscoveryTimeout applies when the configured value is unusable,
	// so a zero in config cannot turn the bounded fetch into an unbounded one.
	defaultDiscoveryTimeout = 10 * time.Second

	// Discovery is retried until the timeout is spent, so a start where the
	// provider is a few seconds behind the application does not kill the
	// process. A refused connection fails instantly, so without the backoff
	// the whole budget would go on a tight loop.
	//
	// It is not a general resilience mechanism: past the timeout the process
	// exits and the orchestrator decides when to try again. A provider that
	// takes half a minute to start -- Keycloak does -- is the orchestrator's
	// problem, which is why compose gives the server a restart policy.
	discoveryBackoff = 500 * time.Millisecond
)

// Verifier turns a raw bearer token into a verified identity.
//
// It is an interface so the api package can be tested without a provider, and
// so a service that authenticates some other way can substitute its own
// implementation without touching the router.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (authx.Identity, error)
}

// Init builds the verifier, returning a nil Verifier when OIDC is disabled.
//
// Unlike otelx.Init, an unreachable provider here is fatal. The asymmetry is
// deliberate and worth stating, because it otherwise reads as an inconsistency
// somebody will tidy away: tracing that fails degrades observability, while
// authentication that fails degrades correctness. A process that starts unable
// to verify a single token is not degraded -- it is an outage that reports
// itself healthy, accepts routing, and rejects every request, which pages the
// application team for an identity provider's incident.
//
// The coupling is confined to startup. The key set is cached in memory and
// refetched only for a key id it has not seen, so a running process keeps
// serving through a provider outage.
func Init(ctx context.Context, cfg config.OIDCConfig) (Verifier, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	if cfg.Issuer == "" {
		return nil, errors.New("oidc.issuer is required when oidc.enabled is true")
	}

	// Checked here rather than left to go-oidc, which reports the same
	// condition as an empty ClientID -- a name that gives no hint that the
	// value wanted is this API's own audience.
	if cfg.Audience == "" {
		return nil, errors.New("oidc.audience is required when oidc.enabled is true")
	}

	timeout := cfg.DiscoveryTimeout
	if timeout <= 0 {
		timeout = defaultDiscoveryTimeout
	}

	// go-oidc takes its http client from the context rather than from a field.
	// Without this, discovery and every later key fetch would use
	// http.DefaultClient, which has no timeout at all.
	ctx = oidc.ClientContext(ctx, &http.Client{Timeout: timeout})

	provider, err := discover(ctx, cfg.Issuer, timeout)
	if err != nil {
		return nil, err
	}

	// The parent context, not the bounded one discovery used: go-oidc keeps
	// this for the life of the key set and reads the http client back off it on
	// every refresh. Cancellation is not the concern -- newRemoteKeySet strips
	// it with context.WithoutCancel -- but the client would be lost.
	tokens := provider.VerifierContext(ctx, &oidc.Config{
		// ClientID is go-oidc's name for the expected audience.
		ClientID:             cfg.Audience,
		SupportedSigningAlgs: supportedSigningAlgs,
		// Every SkipXxx and InsecureXxx field is left false, and none of them
		// is reachable from configuration. See the OIDCConfig doc comment for
		// why they are absent rather than merely defaulted.
	})

	return &verifier{cfg: cfg, tokens: tokens}, nil
}

// discover resolves the provider's metadata, retrying until timeout is spent.
func discover(ctx context.Context, issuer string, timeout time.Duration) (*oidc.Provider, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		provider, err := oidc.NewProvider(ctx, issuer)
		if err == nil {
			return provider, nil
		}

		select {
		case <-ctx.Done():
			return nil, discoveryError(issuer, err)
		case <-time.After(discoveryBackoff):
		}
	}
}

// discoveryError names the issuer, because the usual cause is a value that
// disagrees with the one the provider stamps into its own metadata -- and the
// two are only comparable when both are in the message.
func discoveryError(issuer string, err error) error {
	return fmt.Errorf("failed to discover the oidc provider at %q: %w", issuer, err)
}
