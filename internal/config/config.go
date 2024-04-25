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
	Port                   int `env:"SERVER_PORT, default=8080"`
	ShutdownTimeoutSeconds int `env:"SERVER_SHUTDOWN_TIMEOUT_SECS, default=25"`
}

type AuthorizationConfig struct {
	Audience                  string `env:"JWT_AUDIENCE, default=app-token-issuer"`
	BuildkiteOrganizationSlug string `env:"JWT_BUILDKITE_ORGANIZATION_SLUG, required"`
	IssuerURL                 string `env:"JWT_ISSUER_URL, default=https://agent.buildkite.com"`
	ConfigurationStatic       string `env:"JWT_JWKS_STATIC"`
}

type BuildkiteConfig struct {
	ApiURL string // internal only
	Token  string `env:"BUILDKITE_API_TOKEN, required"`
}

type GithubConfig struct {
	ApiURL         string // internal only
	PrivateKey     string `env:"GITHUB_APP_PRIVATE_KEY, required"`
	ApplicationID  int64  `env:"GITHUB_APP_ID, required"`
	InstallationID int64  `env:"GITHUB_APP_INSTALLATION_ID, required"`
}

func Load(ctx context.Context) (cfg Config, err error) {
	err = envconfig.Process(ctx, &cfg)
	return
}
