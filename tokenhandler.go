package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jamestelfer/ghauth/internal/jwt"
)

type PipelineTokenVendor func(ctx context.Context, claims jwt.BuildkiteClaims) (PipelineRepositoryToken, error)

// Given a pipeline, return the https version of the repository URL
type RepositoryLookup func(ctx context.Context, organizationSlug, pipelineSlug string) (string, error)

// Vend a token for the given repository URL. The URL must be a https URL to a
// GitHub repository that the vendor has permissions to access.
type TokenVendor func(ctx context.Context, repositoryURL string) (string, error)

func handlePostToken(vendor PipelineTokenVendor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// claims must be present from the middleware
		claims := jwt.BuildkiteClaimsFromContext(r.Context())
		if claims == nil {
			requestError(w, http.StatusUnauthorized)
			return
		}

		tokenResponse, err := vendor(r.Context(), *claims)
		if err != nil {
			requestError(w, http.StatusInternalServerError)
			return
		}

		// write the reponse to the client as JSON, supplying the token and URL
		// of the repository it's vended for.
		marshalledResponse, err := json.Marshal(tokenResponse)
		if err != nil {
			requestError(w, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(marshalledResponse)
		if err != nil {
			// record failure to log: trying to respond to the client at this
			// point will likely fail
			fmt.Printf("failed to write response: %v\n", err)
			return
		}
	})
}

func requestError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

type PipelineRepositoryToken struct {
	OrganizationSlug string `json:"organizationSlug"`
	PipelineSlug     string `json:"pipelineSlug"`
	RepositoryURL    string `json:"repositoryUrl"`
	Token            string `json:"token"`
}

func IssueTokenForPipeline(
	repoLookup RepositoryLookup,
	tokenVendor TokenVendor,
) PipelineTokenVendor {
	return func(ctx context.Context, claims jwt.BuildkiteClaims) (PipelineRepositoryToken, error) {
		// use buildkite api to find the repository for the pipeline
		repoURL, err := repoLookup(ctx, claims.OrganizationSlug, claims.PipelineSlug)
		if err != nil {
			return PipelineRepositoryToken{}, fmt.Errorf("could not find repository for pipeline %s: %w", claims.PipelineSlug, err)
		}

		// use the github api to vend a token for the repository
		token, err := tokenVendor(ctx, repoURL)
		if err != nil {
			return PipelineRepositoryToken{}, fmt.Errorf("could not issue token for repository %s: %w", repoURL, err)
		}

		return PipelineRepositoryToken{
			OrganizationSlug: claims.OrganizationSlug,
			PipelineSlug:     claims.PipelineSlug,
			RepositoryURL:    repoURL,
			Token:            token,
		}, nil
	}
}
