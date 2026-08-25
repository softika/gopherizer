package api

import (
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/softika/gopherizer/config"
)

// Supported client IP trust models.
const (
	clientIPFromRemoteAddr = "remote_addr"
	clientIPFromXFF        = "xff"
	clientIPFromHeader     = "header"
)

// clientIPResolver returns the middleware that establishes the caller's address
// for the configured trust model.
//
// The default deliberately ignores proxy headers: a caller controls them, so
// trusting them without a proxy in front lets anyone forge their identity.
// Behind a proxy the opposite failure applies — every client would share the
// proxy's address and land in a single rate limit bucket — which is why the
// deployment has to state which case it is.
func clientIPResolver(cfg config.HTTPConfig) func(http.Handler) http.Handler {
	switch cfg.ClientIP.From {
	case clientIPFromXFF:
		return middleware.ClientIPFromXFFTrustedProxies(cfg.ClientIP.TrustedProxies)

	case clientIPFromHeader:
		if cfg.ClientIP.TrustedHeader == "" {
			// Misconfigured: fall back to the safe, non-spoofable source.
			return middleware.ClientIPFromRemoteAddr
		}
		return middleware.ClientIPFromHeader(cfg.ClientIP.TrustedHeader)

	default:
		return middleware.ClientIPFromRemoteAddr
	}
}

// clientIPKey buckets a request by its resolved client IP.
//
// CanonicalizeIP groups IPv6 addresses by /64, so a client cannot rotate
// through its own prefix to reset the limit.
//
// The fallback covers the case where no resolver ran. It uses the socket peer,
// which net/http sets and a caller cannot forge; a header would be spoofable.
// Behind a proxy this buckets every caller together, so the router always
// installs clientIPResolver ahead of the limiter to make the trust model
// explicit rather than leaving it to this fallback.
func clientIPKey(r *http.Request) (string, error) {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return httprate.CanonicalizeIP(ip), nil
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return httprate.CanonicalizeIP(host), nil
}
