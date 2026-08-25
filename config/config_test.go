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
