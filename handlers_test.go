package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/jamestelfer/chinmina-bridge/internal/jwt"
	"github.com/jamestelfer/chinmina-bridge/internal/vendor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultExpiry = time.Date(2024, time.May, 7, 17, 59, 36, 0, time.UTC)

func TestHandlePostToken_RequiresClaims(t *testing.T) {
	req, err := http.NewRequest("POST", "/token", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	handler := handlePostToken(nil)

	assert.PanicsWithValue(t, "Buildkite claims not present in context, likely used outside of the JWT middleware", func() {
		handler.ServeHTTP(rr, req)
	})
}

func TestHandlePostToken_ReturnsTokenOnSuccess(t *testing.T) {
	tokenVendor := tv("expected-token-value")

	ctx := context.Background()
	ctx = jwt.ContextWithClaims(ctx, &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Issuer: "https://buildkite.com",
		},
		CustomClaims: &jwt.BuildkiteClaims{
			OrganizationSlug: "organization-slug",
			PipelineSlug:     "pipeline-slug",
		},
	})

	req, err := http.NewRequest("POST", "/token", nil)
	require.NoError(t, err)

	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	// act
	handler := handlePostToken(tokenVendor)
	handler.ServeHTTP(rr, req)

	// assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	respBody := vendor.PipelineRepositoryToken{}
	err = json.Unmarshal(rr.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, &vendor.PipelineRepositoryToken{
		Token:            "expected-token-value",
		Expiry:           defaultExpiry,
		OrganizationSlug: "organization-slug",
		PipelineSlug:     "pipeline-slug",
	}, &respBody)
}

func TestHandlePostToken_ReturnsFailureOnVendorFailure(t *testing.T) {
	tokenVendor := tvFails(errors.New("vendor failure"))

	ctx := context.Background()
	ctx = jwt.ContextWithClaims(ctx, &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Issuer: "https://buildkite.com",
		},
		CustomClaims: &jwt.BuildkiteClaims{
			OrganizationSlug: "organization-slug",
			PipelineSlug:     "pipeline-slug",
		},
	})

	req, err := http.NewRequest("POST", "/token", nil)
	require.NoError(t, err)

	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	// act
	handler := handlePostToken(tokenVendor)
	handler.ServeHTTP(rr, req)

	// assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// important to know that internal details aren't part of the error response
	assert.Equal(t, "Internal Server Error\n", rr.Body.String())
}

func tv(token string) vendor.PipelineTokenVendor {
	return vendor.PipelineTokenVendor(func(_ context.Context, claims jwt.BuildkiteClaims, repoUrl string) (*vendor.PipelineRepositoryToken, error) {
		return &vendor.PipelineRepositoryToken{
			Token:            token,
			Expiry:           defaultExpiry,
			PipelineSlug:     claims.PipelineSlug,
			OrganizationSlug: claims.OrganizationSlug,
			RepositoryURL:    repoUrl,
		}, nil
	})
}

func tvFails(err error) vendor.PipelineTokenVendor {
	return vendor.PipelineTokenVendor(func(_ context.Context, claims jwt.BuildkiteClaims, repoUrl string) (*vendor.PipelineRepositoryToken, error) {
		return nil, err
	})
}
