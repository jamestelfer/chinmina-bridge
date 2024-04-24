package buildkite

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/buildkite/go-buildkite/v3/buildkite"
	"github.com/jamestelfer/ghauth/internal/config"
)

type PipelineLookup struct {
	client *buildkite.Client
}

func New(cfg config.BuildkiteConfig) PipelineLookup {
	transport, err := buildkite.NewTokenConfig(cfg.Token, false)
	if err != nil {
		log.Fatalf("client config failed: %s", err)
	}

	client := buildkite.NewClient(transport.Client())

	if cfg.ApiURL != "" {
		url, _ := url.Parse(cfg.ApiURL)
		transport.APIHost = url.Host
		client.BaseURL, _ = url.Parse(cfg.ApiURL)
	}

	return PipelineLookup{
		client,
	}
}

func (p PipelineLookup) RepositoryLookup(ctx context.Context, organizationSlug, pipelineSlug string) (string, error) {
	pipeline, _, err := p.client.Pipelines.Get(organizationSlug, pipelineSlug)
	if err != nil {
		return "", fmt.Errorf("failed to get pipeline called %s/%s: %w", organizationSlug, pipelineSlug, err)

	}

	repo := pipeline.Repository
	if repo == nil {
		return "", fmt.Errorf("no configured repository for pipeline %s/%s", organizationSlug, pipelineSlug)
	}

	return *repo, nil
}
