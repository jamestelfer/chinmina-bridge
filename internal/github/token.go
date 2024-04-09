package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v61/github"
	"github.com/jamestelfer/ghauth/internal/config"
)

type Client struct {
	client         *github.Client
	installationID int64
}

func New(cfg config.GithubConfig) (Client, error) {

	// Create a transport using the JWT authentication method. The endpoints
	// we're calling require this method.
	appInstallationTransport, err := ghinstallation.NewAppsTransport(
		http.DefaultTransport,
		cfg.ApplicationID,
		[]byte(cfg.PrivateKey),
	)
	if err != nil {
		return Client{}, fmt.Errorf("could not create github transport: %w", err)
	}

	// Create a client for use with the application credentials. This client
	// will be used concurrently.
	client := github.NewClient(
		&http.Client{
			Transport: appInstallationTransport,
		},
	)

	return Client{
		client,
		cfg.InstallationID,
	}, nil
}

func (c Client) CreateAccessToken(ctx context.Context, repositoryURL string) (string, error) {
	u, err := url.Parse(repositoryURL)
	if err != nil {
		return "", err
	}

	qualifiedIdentifier, _ := strings.CutSuffix(u.Path, ".git")
	_, repoName, _ := strings.Cut(qualifiedIdentifier[1:], "/")

	tok, _, err := c.client.Apps.CreateInstallationToken(ctx, c.installationID,
		&github.InstallationTokenOptions{
			Repositories: []string{repoName},
			Permissions: &github.InstallationPermissions{
				Contents: github.String("read"),
			},
		},
	)
	if err != nil {
		return "", err
	}

	return tok.GetToken(), nil
}
