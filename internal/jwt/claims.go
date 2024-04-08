package jwt

import (
	"context"
	"fmt"
	"strings"

	"github.com/auth0/go-jwt-middleware/v2/validator"
)

// BuildkiteClaims define the additional claims that Builkite includes in the
// JWT.
//
// See: https://buildkite.com/docs/agent/v3/cli-oidc#claims
type BuildkiteClaims struct {
	// Audience is a registered claim, so will be returned regardless. Adding it
	// here however allows us to validate that it's present with less ceremony.
	Audience         string `json:"aud"`
	OrganizationSlug string `json:"organization_slug"`
	PipelineSlug     string `json:"pipeline_slug"`
	BuildNumber      string `json:"build_number"`
	BuildBranch      string `json:"build_branch"`
	BuildTag         string `json:"build_tag"`
	BuildCommit      string `json:"build_commit"`
	StepKey          string `json:"step_key"`
	JobId            string `json:"job_id"`
	AgentId          string `json:"agent_id"`

	expectedOrganizationSlug string
}

// Validate ensures that the expected claims are present in the token, and that
// the organization slug matches the configured value.
func (c *BuildkiteClaims) Validate(ctx context.Context) error {

	fields := [][]string{
		{"aud", c.Audience},
		{"organization_slug", c.OrganizationSlug},
		{"pipeline_slug", c.PipelineSlug},
		{"build_number", c.BuildNumber},
		{"build_branch", c.BuildBranch},
		{"build_commit", c.BuildCommit},
		{"step_key", c.StepKey},
		{"job_id", c.JobId},
		{"agent_id", c.AgentId},
	}

	missing := []string{}

	for _, pair := range fields {
		if pair[1] == "" {
			missing = append(missing, pair[0])
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing expected claim(s): %s", strings.Join(missing, ", "))
	}

	if c.expectedOrganizationSlug != "" && c.expectedOrganizationSlug != c.OrganizationSlug {
		return fmt.Errorf("expecting token issued for organization %s", c.expectedOrganizationSlug)
	}

	return nil
}

// buildkiteCustomClaims sets up OIDC custom claims for a Buildkite-issued JWT.
func buildkiteCustomClaims(expectedOrganizationSlug string) func() validator.CustomClaims {
	return func() validator.CustomClaims {
		return &BuildkiteClaims{
			expectedOrganizationSlug: expectedOrganizationSlug,
		}
	}
}
