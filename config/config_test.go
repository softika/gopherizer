package config

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfigHasNoOrphanKeys fails when default.toml declares a key that
// no Config field consumes.
//
// Viper discards unknown keys silently, so such a key reads as configured while
// doing nothing — which is how an [http.auth] secret sat in the file unbound.
func TestDefaultConfigHasNoOrphanKeys(t *testing.T) {
	t.Parallel()

	v := viper.New()
	v.SetConfigType("toml")

	file, err := configFile.Open("default.toml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	require.NoError(t, v.ReadConfig(file))

	err = v.Unmarshal(new(Config), func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnused = true
	})

	assert.NoError(t, err, "every key in default.toml must bind to a Config field")
}

// TestNewLoadsDefaults covers the values the server depends on at boot.
func TestNewLoadsDefaults(t *testing.T) {
	cfg, err := New()
	require.NoError(t, err)

	assert.Equal(t, "gopherizer", cfg.App.Name)
	assert.NotEmpty(t, cfg.Http.Port)

	assert.NotZero(t, cfg.Http.ReadHeaderTimeout, "slow-header protection must be configured")
	assert.NotZero(t, cfg.Http.IdleTimeout)

	assert.Positive(t, cfg.Http.RateLimit.Requests, "rate limiting must be configured by default")
	assert.Positive(t, cfg.Http.RateLimit.Window)

	assert.NotEmpty(t, cfg.Http.Cors.Origins, "cors origins must be configured")
	assert.NotEmpty(t, cfg.Http.Cors.Methods)
}

// TestNewRejectsUnknownLogLevel proves a typo fails at startup rather than
// silently falling back to a level nobody chose.
//
// It does not run in parallel: it mutates the environment, and New reads the
// process-wide viper instance.
func TestNewRejectsUnknownLogLevel(t *testing.T) {
	t.Setenv("APP_LOG_LEVEL", "verbose")

	_, err := New()

	assert.Error(t, err, "an unrecognised log level must fail validation")
}

func TestNewAcceptsLogLevelOverride(t *testing.T) {
	t.Setenv("APP_LOG_LEVEL", "warn")

	cfg, err := New()

	require.NoError(t, err)
	assert.Equal(t, "warn", cfg.App.LogLevel)
}

// TestNewDefaultsLogLevelToEmpty pins that the shipped default derives the
// level from environment rather than hardcoding one.
func TestNewDefaultsLogLevelToEmpty(t *testing.T) {
	cfg, err := New()

	require.NoError(t, err)
	assert.Empty(t, cfg.App.LogLevel)
}

// The shipped defaults must leave authentication off, or the template stops
// running out of the box and every existing test needs an identity provider.
//
// It does not run in parallel: New reads the process-wide viper instance.
func TestOidcIsDisabledByDefault(t *testing.T) {
	cfg, err := New()

	require.NoError(t, err)
	assert.False(t, cfg.Oidc.Enabled, "authentication must be opt-in")
	assert.Equal(t, "scope", cfg.Oidc.ScopeClaim)
	assert.Equal(t, "roles", cfg.Oidc.RolesClaim,
		"the default must stay vendor neutral; the Keycloak path belongs with the Keycloak profile")
	assert.Positive(t, cfg.Oidc.DiscoveryTimeout, "discovery must be bounded")
	assert.False(t, cfg.Oidc.RequireTypedAccessToken,
		"rfc 9068 is opt-in because not every provider stamps the header")
}

// Every key has to be reachable by environment variable, because viper only
// binds the ones already present in the embedded file -- which is exactly how
// the unbound [http.auth] secret this package's guard test records came about.
func TestOidcSettingsBindFromTheEnvironment(t *testing.T) {
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://issuer.example/realms/test")
	t.Setenv("OIDC_AUDIENCE", "gopherizer-api")
	t.Setenv("OIDC_ROLES_CLAIM", "realm_access.roles")
	t.Setenv("OIDC_REQUIRE_TYPED_ACCESS_TOKEN", "true")

	cfg, err := New()

	require.NoError(t, err)
	assert.True(t, cfg.Oidc.Enabled)
	assert.Equal(t, "https://issuer.example/realms/test", cfg.Oidc.Issuer)
	assert.Equal(t, "gopherizer-api", cfg.Oidc.Audience)
	assert.Equal(t, "realm_access.roles", cfg.Oidc.RolesClaim)
	assert.True(t, cfg.Oidc.RequireTypedAccessToken)
}
