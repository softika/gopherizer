package config

import (
	"embed"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

//go:embed default.toml
var configFile embed.FS

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Http     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
}

func New() (*Config, error) {
	viper.SetConfigType("toml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()

	file, err := configFile.Open("default.toml")
	if err != nil {
		slog.Error("failed reading config file", "error", err.Error())
		return nil, err
	}

	if err = viper.ReadConfig(file); err != nil {
		slog.Error("failed reading config file", "error", err.Error())
		return nil, err
	}

	config := new(Config)
	if err = viper.Unmarshal(config); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err = validate.Struct(config); err != nil {
		return nil, err
	}
	return config, nil
}

type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	Version     string `mapstructure:"version" validate:"required"`
	// LogLevel overrides the level implied by Environment. Empty derives it,
	// so a deployment only sets this when it wants something other than the
	// default. An unrecognised value fails at startup rather than silently
	// falling back.
	LogLevel string `mapstructure:"log_level" validate:"omitempty,oneof=debug info warn error"`
}

type HTTPConfig struct {
	Host              string        `mapstructure:"host"`
	Port              string        `mapstructure:"port" validate:"required"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	// RequestTimeout bounds how long a single request may take. It sits inside
	// WriteTimeout and outside the database statement timeout, so the most
	// specific limit fires first and the error names the real cause.
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	// MaxBodyBytes caps a request body. Zero disables the limit, which leaves
	// the process open to a trivial memory exhaustion.
	MaxBodyBytes int64 `mapstructure:"max_body_bytes" validate:"gte=0"`
	Cors         struct {
		Origins string `mapstructure:"origins"`
		Methods string `mapstructure:"methods"`
		Headers string `mapstructure:"headers"`
	} `mapstructure:"cors"`
	RateLimit struct {
		// Requests allowed per Window, per client. Zero disables limiting.
		Requests int           `mapstructure:"requests"`
		Window   time.Duration `mapstructure:"window"`
	} `mapstructure:"rate_limit"`
	// ClientIP declares the trust model used to resolve a caller's address.
	// Getting this wrong either lumps every client behind a proxy into one
	// bucket, or lets callers spoof their identity via a header.
	ClientIP struct {
		// From selects the resolver: "remote_addr" (default), "xff", "header".
		From string `mapstructure:"from"`
		// TrustedProxies is the number of reverse proxies in front of the app,
		// used when From is "xff".
		TrustedProxies int `mapstructure:"trusted_proxies"`
		// TrustedHeader is the proxy-set header, used when From is "header".
		TrustedHeader string `mapstructure:"trusted_header"`
	} `mapstructure:"client_ip"`
	Metrics struct {
		// Enabled exposes the Prometheus endpoint. Keep it off the public
		// listener in production, or restrict it at the ingress.
		Enabled bool   `mapstructure:"enabled"`
		Path    string `mapstructure:"path"`
	} `mapstructure:"metrics"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host" validate:"required"`
	Port            string `mapstructure:"port" validate:"required"`
	DBName          string `mapstructure:"dbname" validate:"required"`
	Password        string `mapstructure:"password" validate:"required"`
	User            string `mapstructure:"user" validate:"required"`
	SSLModeDisabled bool   `mapstructure:"sslmode_disabled"`

	// MaxConns bounds the pool. Left unset, pgx derives it from the machine's
	// CPU count, so the same image silently gets a different ceiling on every
	// host size -- and the database's own connection limit is a shared budget
	// that no single service should size by accident.
	MaxConns int32 `mapstructure:"max_conns" validate:"gte=0"`
	// MinConns keeps connections warm so a burst does not pay to reconnect.
	MinConns int32 `mapstructure:"min_conns" validate:"gte=0"`
	// MaxConnLifetime retires connections regardless of health, which is what
	// lets a failover or a rotated credential take effect without a restart.
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	// MaxConnIdleTime releases connections the pool is no longer using.
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	// HealthCheckPeriod is how often idle connections are verified.
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
	// StatementTimeout is applied by Postgres itself, so a runaway query is
	// cancelled server-side rather than holding a pooled connection until the
	// HTTP write deadline. Zero disables it.
	//
	// It applies to migrations too, since they share this configuration. Raise
	// it for a deployment that runs long ones.
	StatementTimeout time.Duration `mapstructure:"statement_timeout"`
}

// TracingConfig configures OpenTelemetry trace export.
//
// Disabled by default: the template must run, and its tests must pass, without
// a collector listening anywhere.
type TracingConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Endpoint is the OTLP/HTTP receiver as host:port, with no scheme.
	Endpoint string `mapstructure:"endpoint"`
	// Insecure sends over plain HTTP. Acceptable to a collector on localhost,
	// wrong across a network.
	Insecure bool `mapstructure:"insecure"`
	// SampleRatio is the head sampling ratio for traces started here. Traces
	// started upstream follow the caller's decision.
	SampleRatio float64 `mapstructure:"sample_ratio" validate:"gte=0,lte=1"`
}
