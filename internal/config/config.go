package config

import (
	"time"
)

type Config struct {
	Addr    string
	Timeout time.Duration
}

func Load() *Config {
	return &Config{
		Addr:    "127.0.0.1:5004",
		Timeout: 10 * time.Second,
	}
}
