package config

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"

	"github.com/softika/slogging"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Http     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

func New() (*Config, error) {
	viper.AddConfigPath("config")
	viper.SetConfigName("config")
	viper.SetConfigType("ini")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slogging.Slogger().Error("Error reading config file", "error", err.Error())
		return nil, err
	}

	config := new(Config)
	err := viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, err
	}
	return config, nil
}

type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	Version     string `mapstructure:"version" validate:"required"`
}

type AuthConfig struct {
	Secret   string        `mapstructure:"secret"`
	TokenExp time.Duration `mapstructure:"token_exp"`
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
