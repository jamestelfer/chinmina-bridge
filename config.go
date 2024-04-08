package main

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Server        ServerConfig
}

type ServerConfig struct {
	Port int `env:"PORT, default=8080"`
}

func loadConfig(ctx context.Context) (cfg Config, err error) {
	err = envconfig.Process(ctx, &cfg)
	return
}
