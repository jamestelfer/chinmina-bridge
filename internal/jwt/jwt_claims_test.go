package jwt

import (
	"context"
	"testing"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/stretchr/testify/assert"
)

type UselessCustomClaims struct{}

func (*UselessCustomClaims) Validate(ctx context.Context) error {
	return nil
}

func TestContextClaims(t *testing.T) {

	cases := []struct {
		name   string
		claims *validator.ValidatedClaims
	}{
		{
			name: "no claims",
		},
		{
			name: "empty claims",
			claims: &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{},
				CustomClaims:     nil,
			},
		},
		{
			name: "registered claims",
			claims: &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Audience: []string{"audience"},
					Subject:  "subject",
					Issuer:   "issuer",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, tc.claims)
			actual := ClaimsFromContext(ctx)
			assert.Equal(t, tc.claims, actual)
		})
	}
}

func TestCustomClaims(t *testing.T) {
	cases := []struct {
		name                 string
		claims               *validator.ValidatedClaims
		expectedCustomClaims *BuildkiteClaims
	}{
		{
			name: "no claims",
		},
		{
			name: "unknown claims",
			claims: &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Audience: []string{"audience"},
					Subject:  "subject",
					Issuer:   "issuer",
				},
				CustomClaims: &UselessCustomClaims{},
			},
		},
		{
			name: "unknown claims",
			claims: &validator.ValidatedClaims{
				RegisteredClaims: validator.RegisteredClaims{
					Audience: []string{"audience"},
					Subject:  "subject",
					Issuer:   "issuer",
				},
				CustomClaims: &BuildkiteClaims{
					OrganizationSlug: "expected-organization",
					PipelineSlug:     "expected-pipeline",
				},
			},
			expectedCustomClaims: &BuildkiteClaims{
				OrganizationSlug: "expected-organization",
				PipelineSlug:     "expected-pipeline",
			},
		}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, tc.claims)
			actual := BuildkiteClaimsFromContext(ctx)
			assert.Equal(t, tc.expectedCustomClaims, actual)
		})
	}
}
