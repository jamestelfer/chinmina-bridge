package jwt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"

	"github.com/jamestelfer/ghauth/internal/config"
)

// VerificationMiddleware returns HTTP middleware that verifies the JWT and
// enforces the validity claims. The retrieved claims are set on the request
// context and can be retrieved by calling jwt.ClaimsFromContext(ctx).
func VerificationMiddleware(cfg config.AuthorizationConfig) (func(http.Handler) http.Handler, error) {
	issuerURL, err := url.Parse(cfg.ConfigurationURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the issuer URL: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	// the validator is used by the middleware to check the JWT signature and claims
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		// Buildkite only uses RSA at present
		validator.RS256,
		issuerURL.String(),
		[]string{cfg.Audience},
		validator.WithAllowedClockSkew(5*time.Second), // this could be configurable
		validator.WithCustomClaims(
			buildkiteCustomClaims(cfg.BuildkiteOrganizationSlug),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set up the validator: %v", err)
	}

	return jwtmiddleware.New(jwtValidator.ValidateToken).CheckJWT, nil
}

// ClaimsFromContext returns the validated claims from the context as set by the
// JWT middleware. This will return nil if the context data is not set. This
// should be regarded as an error for handlers that expect the claims to be
// present.
func ClaimsFromContext(ctx context.Context) *validator.ValidatedClaims {
	claims, _ := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	return claims
}
