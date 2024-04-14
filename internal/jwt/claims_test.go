package jwt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildkiteClaims_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		claims := &BuildkiteClaims{
			OrganizationSlug:         "org",
			PipelineSlug:             "pipeline",
			PipelineID:               "pipeline_uuid",
			BuildNumber:              123,
			BuildBranch:              "main",
			BuildCommit:              "abc123",
			StepKey:                  "step1",
			JobId:                    "job1",
			AgentId:                  "agent1",
			expectedOrganizationSlug: "org",
		}

		err := claims.Validate(context.Background())

		assert.NoError(t, err)
	})

	t.Run("missing claims", func(t *testing.T) {
		claims := &BuildkiteClaims{}

		err := claims.Validate(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing expected claim(s)")
	})

	t.Run("wrong org", func(t *testing.T) {
		claims := &BuildkiteClaims{
			PipelineSlug: "pipeline",
			PipelineID:   "pipeline_uuid",
			BuildNumber:  123,
			BuildBranch:  "main",
			BuildCommit:  "abc123",
			StepKey:      "step1",
			JobId:        "job1",
			AgentId:      "agent1",

			OrganizationSlug:         "wrong",
			expectedOrganizationSlug: "right",
		}

		err := claims.Validate(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expecting token issued for organization")
	})
}
