package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Server        ServerConfig
	Authorization AuthorizationConfig
}

type ServerConfig struct {
	Port int `env:"PORT, default=8080"`
}

type AuthorizationConfig struct {
	Audience                  string `env:"JWT_AUDIENCE, default=app-token-issuer"`
	BuildkiteOrganizationSlug string `env:"JWT_BUILDKITE_ORGANIZATION_SLUG, required"`
	ConfigurationURL          string `env:"JWT_JWKS_URL, default=https://buildkite.com/.well-known/jwks"`
}

func Load(ctx context.Context) (cfg Config, err error) {
	err = envconfig.Process(ctx, &cfg)
	return
}
