package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-jose/go-jose/v4"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/rs/zerolog/log"

	"github.com/jamestelfer/chinmina-bridge/internal/config"
)

// Middleware returns HTTP middleware that verifies the JWT and
// enforces the validity claims. The retrieved claims are set on the request
// context and can be retrieved by calling jwt.ClaimsFromContext(ctx).
func Middleware(cfg config.AuthorizationConfig, options ...jwtmiddleware.Option) (func(http.Handler) http.Handler, error) {
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

	// wrap the standard validator with additional validaton that ensures the
	// core claims (including validity periods) are present
	tokenValidator := registeredClaimsValidator(jwtValidator.ValidateToken)

	return jwtmiddleware.New(tokenValidator, options...).CheckJWT, nil
}

func LogErrorHandler() jwtmiddleware.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		log.Warn().
			Err(err).
			Msg("JWT decode failure")

		jwtmiddleware.DefaultErrorHandler(w, r, err)
	}
}

// ContextWithClaims returns a new context.Context with the provided validated claims
// added to it. This is primarily for test usage
func ContextWithClaims(ctx context.Context, claims *validator.ValidatedClaims) context.Context {
	return context.WithValue(ctx, jwtmiddleware.ContextKey{}, claims)
}

// ClaimsFromContext returns the validated claims from the context as set by the
// JWT middleware. This will return nil if the context data is not set. This
// should be regarded as an error for handlers that expect the claims to be
// present.
func ClaimsFromContext(ctx context.Context) *validator.ValidatedClaims {
	claims, _ := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	return claims
}

// Get the custom Buildkite claims from the context, as added by the JWT
// middleware. This will return nil if the claims are not present.
func BuildkiteClaimsFromContext(ctx context.Context) *BuildkiteClaims {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}

	bkClaims, _ := claims.CustomClaims.(*BuildkiteClaims)

	return bkClaims
}

func RequireBuildkiteClaimsFromContext(ctx context.Context) BuildkiteClaims {
	c := BuildkiteClaimsFromContext(ctx)
	if c == nil {
		panic("Buildkite claims not present in context, likely used outside of the JWT middleware")
	}

	return *c
}

type KeyFunc = func(ctx context.Context) (any, error)

func remoteJWKS(cfg config.AuthorizationConfig) (url.URL, KeyFunc, error) {
	issuerURL, err := url.Parse(cfg.IssuerURL)
	if err != nil {
		return url.URL{}, nil, fmt.Errorf("failed to parse the issuer URL: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	return *issuerURL, provider.KeyFunc, nil
}

func staticJWKS(cfg config.AuthorizationConfig) (url.URL, KeyFunc, error) {
	issuerURL, err := url.Parse(cfg.IssuerURL)
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
