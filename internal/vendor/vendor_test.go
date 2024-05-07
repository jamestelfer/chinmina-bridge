package vendor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jamestelfer/chinmina-bridge/internal/jwt"
	"github.com/jamestelfer/chinmina-bridge/internal/vendor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVendor_FailWhenPipelineLookupFails(t *testing.T) {
	repoLookup := vendor.RepositoryLookup(func(ctx context.Context, org string, pipeline string) (string, error) {
		return "", errors.New("pipeline not found")
	})
	v := vendor.New(repoLookup, nil)

	_, err := v(context.Background(), jwt.BuildkiteClaims{}, "repo-url")
	require.ErrorContains(t, err, "could not find repository for pipeline")
}

func TestVendor_SuccessfulNilOnRepoMismatch(t *testing.T) {
	repoLookup := vendor.RepositoryLookup(func(ctx context.Context, org string, pipeline string) (string, error) {
		return "repo-url-mismatch", nil
	})
	v := vendor.New(repoLookup, nil)

	// when there is a difference between the requested pipeline (by Git
	// generally) and the repo associated with the pipeline, return success but
	// empty. This indicates that there are not credentials that can be issued.

	tok, err := v(
		context.Background(),
		jwt.BuildkiteClaims{PipelineID: "pipeline-id", PipelineSlug: "pipeline-slug", OrganizationSlug: "organization-slug"},
		"repo-url",
	)
	assert.NoError(t, err)
	assert.Nil(t, tok)
}

func TestVendor_FailsWhenTokenVendorFails(t *testing.T) {
	repoLookup := vendor.RepositoryLookup(func(ctx context.Context, org string, pipeline string) (string, error) {
		return "repo-url", nil
	})
	tokenVendor := vendor.TokenVendor(func(ctx context.Context, repositoryURL string) (string, time.Time, error) {
		return "", time.Time{}, errors.New("token vendor failed")
	})
	v := vendor.New(repoLookup, tokenVendor)

	tok, err := v(context.Background(), jwt.BuildkiteClaims{PipelineID: "pipeline-id", PipelineSlug: "pipeline-slug", OrganizationSlug: "organization-slug"}, "repo-url")
	assert.ErrorContains(t, err, "token vendor failed")
	assert.Nil(t, tok)
}

func TestVendor_SucceedsWithTokenWhenPossible(t *testing.T) {
	vendedDate := time.Date(1970, 1, 1, 0, 0, 10, 0, time.UTC)

	repoLookup := vendor.RepositoryLookup(func(ctx context.Context, org string, pipeline string) (string, error) {
		return "repo-url", nil
	})
	tokenVendor := vendor.TokenVendor(func(ctx context.Context, repositoryURL string) (string, time.Time, error) {
		return "vended-token-value", vendedDate, nil
	})
	v := vendor.New(repoLookup, tokenVendor)

	tok, err := v(context.Background(), jwt.BuildkiteClaims{PipelineID: "pipeline-id", PipelineSlug: "pipeline-slug", OrganizationSlug: "organization-slug"}, "repo-url")
	assert.NoError(t, err)
	assert.Equal(t, tok, &vendor.PipelineRepositoryToken{
		Token:            "vended-token-value",
		Expiry:           vendedDate,
		OrganizationSlug: "organization-slug",
		PipelineSlug:     "pipeline-slug",
		RepositoryURL:    "repo-url",
	})
}
