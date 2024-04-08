package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"gopkg.in/go-jose/go-jose.v2"

	"github.com/jamestelfer/ghauth/internal/config"
)

// VerificationMiddleware returns HTTP middleware that verifies the JWT and
// enforces the validity claims. The retrieved claims are set on the request
// context and can be retrieved by calling jwt.ClaimsFromContext(ctx).
func VerificationMiddleware(cfg config.AuthorizationConfig) (func(http.Handler) http.Handler, error) {
	// allow for static configuration when testing
	jwksConfig := remoteJWKS
	if cfg.ConfigurationStatic != "" {
		jwksConfig = staticJWKS
	}

	url, keyFunc, err := jwksConfig(cfg)
	if err != nil {
		return nil, err
	}

	// the validator is used by the middleware to check the JWT signature and claims
	jwtValidator, err := validator.New(
		keyFunc,
		// Buildkite only uses RSA at present
		validator.RS256,
		url.String(),
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

type KeyFunc = func(ctx context.Context) (any, error)

func remoteJWKS(cfg config.AuthorizationConfig) (url.URL, KeyFunc, error) {
	issuerURL, err := url.Parse(cfg.ConfigurationURL)
	if err != nil {
		return url.URL{}, nil, fmt.Errorf("failed to parse the issuer URL: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	return *issuerURL, provider.KeyFunc, nil
}

func staticJWKS(cfg config.AuthorizationConfig) (url.URL, KeyFunc, error) {
	issuerURL, err := url.Parse(cfg.ConfigurationURL)
	if err != nil {
		return url.URL{}, nil, fmt.Errorf("failed to parse the issuer URL: %w", err)
	}

	var jwks jose.JSONWebKeySet
	if err := json.Unmarshal([]byte(cfg.ConfigurationStatic), &jwks); err != nil {
		return url.URL{}, nil, fmt.Errorf("could not decode jwks: %w", err)
	}

	keyFunc := func(_ context.Context) (any, error) { return jwks, nil }

	return *issuerURL, keyFunc, nil
}
