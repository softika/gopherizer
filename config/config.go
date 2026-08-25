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
}

type HTTPConfig struct {
	Host              string        `mapstructure:"host"`
	Port              string        `mapstructure:"port" validate:"required"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	Cors              struct {
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
}
