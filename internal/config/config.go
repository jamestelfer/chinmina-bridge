package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Authorization AuthorizationConfig
	Buildkite     BuildkiteConfig
	Github        GithubConfig
	Server        ServerConfig
}

type ServerConfig struct {
	Port int `env:"PORT, default=8080"`
}

type AuthorizationConfig struct {
	Audience                  string `env:"JWT_AUDIENCE, default=app-token-issuer"`
	BuildkiteOrganizationSlug string `env:"JWT_BUILDKITE_ORGANIZATION_SLUG, required"`
	ConfigurationURL          string `env:"JWT_JWKS_URL, default=https://buildkite.com/.well-known/jwks"`
	ConfigurationStatic       string `env:"JWT_JWKS_STATIC"`
}

type BuildkiteConfig struct {
	Token string `env:"BUILDKITE_API_TOKEN, required"`
}

type GithubConfig struct {
	PrivateKey     string `env:"GITHUB_APP_PRIVATE_KEY, required"`
	ApplicationID  int64  `env:"GITHUB_APP_ID, required"`
	InstallationID int64  `env:"GITHUB_APP_INSTALLATION_ID, required"`
}

func Load(ctx context.Context) (cfg Config, err error) {
	err = envconfig.Process(ctx, &cfg)
	return
}
