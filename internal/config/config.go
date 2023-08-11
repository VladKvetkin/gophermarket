package config

import (
	"flag"
	"net/url"

	"github.com/caarlos0/env/v8"
)

type Config struct {
	Address              string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func NewConfig() (Config, error) {
	config := Config{}

	config.parseFlags()

	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}

	if err := config.validateConfig(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *Config) parseFlags() {
	flag.StringVar(&c.Address, "a", c.Address, "Service address")
	flag.StringVar(&c.DatabaseURI, "d", c.DatabaseURI, "Database URI")
	flag.StringVar(&c.AccrualSystemAddress, "r", c.AccrualSystemAddress, "Accrual system address")

	flag.Parse()
}

func (c *Config) validateConfig() error {
	for _, URI := range []string{c.Address, c.AccrualSystemAddress} {
		_, err := url.ParseRequestURI(URI)
		if err != nil {
			return err
		}
	}

	return nil
}
