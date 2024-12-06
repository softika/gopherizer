package config

import (
	"embed"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

//go:embed default.ini
var configFile embed.FS

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Http     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
}

func New() (*Config, error) {
	viper.SetConfigType("ini")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()

	file, err := configFile.Open("default.ini")
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
	Host         string        `mapstructure:"host"`
	Port         string        `mapstructure:"port" validate:"required"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	Cors         struct {
		Origins string `mapstructure:"origins"`
		Methods string `mapstructure:"methods"`
		Headers string `mapstructure:"headers"`
	} `mapstructure:"cors"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host" validate:"required"`
	Port            string `mapstructure:"port" validate:"required"`
	DBName          string `mapstructure:"dbname" validate:"required"`
	Password        string `mapstructure:"password" validate:"required"`
	User            string `mapstructure:"user" validate:"required"`
	SSLModeDisabled bool   `mapstructure:"sslmode_disabled"`
}
