package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/cors"

	"github.com/softika/gopherizer/config"
)

// corsMaxAge caches preflight results for five minutes, the ceiling most
// browsers honour.
const corsMaxAge = 300

// corsMiddleware builds the CORS middleware, or nil when no origins are
// configured.
//
// Credentials are never allowed alongside a wildcard origin: browsers reject
// that combination outright, and honouring it would expose authenticated
// responses to any site. Narrow Origins to real hosts before enabling them.
func corsMiddleware(cfg config.HTTPConfig) func(http.Handler) http.Handler {
	origins := splitList(cfg.Cors.Origins)
	if len(origins) == 0 {
		return nil
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   splitList(cfg.Cors.Methods),
		AllowedHeaders:   splitList(cfg.Cors.Headers),
		AllowCredentials: !slices.Contains(origins, "*"),
		MaxAge:           corsMaxAge,
	})
}

// splitList parses a comma-separated config value, dropping empty entries.
func splitList(v string) []string {
	var out []string

	for _, part := range strings.Split(v, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}
