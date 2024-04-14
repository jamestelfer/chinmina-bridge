package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jamestelfer/ghauth/internal/credentialhandler"
	"github.com/jamestelfer/ghauth/internal/jwt"
	"github.com/maypok86/otter"
	"github.com/rs/zerolog/log"
)

type PipelineTokenVendor func(ctx context.Context, claims jwt.BuildkiteClaims, repo string) (*PipelineRepositoryToken, error)

// Given a pipeline, return the https version of the repository URL
type RepositoryLookup func(ctx context.Context, organizationSlug, pipelineSlug string) (string, error)

// Vend a token for the given repository URL. The URL must be a https URL to a
// GitHub repository that the vendor has permissions to access.
type TokenVendor func(ctx context.Context, repositoryURL string) (string, time.Time, error)

func handlePostToken(vendor PipelineTokenVendor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// claims must be present from the middleware
		claims := jwt.BuildkiteClaimsFromContext(r.Context())
		if claims == nil {
			requestError(w, http.StatusUnauthorized)
			return
		}

		tokenResponse, err := vendor(r.Context(), *claims, "")
		if err != nil {
			log.Info().Msgf("token creation failed %v\n", err)
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
			log.Info().Msgf("failed to write response: %v\n", err)
			return
		}
	})
}

func handlePostGitCredentials(vendor PipelineTokenVendor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure that the request body is fully read prior to returning. This
		// avoids issues with blocked connections and connection reuse.
		defer func() { io.Copy(io.Discard, r.Body) }()

		// claims must be present from the middleware
		claims := jwt.BuildkiteClaimsFromContext(r.Context())
		if claims == nil {
			requestError(w, http.StatusUnauthorized)
			return
		}

		requestedRepo, err := credentialhandler.ReadProperties(r.Body)
		if err != nil {
			log.Info().Msgf("read repository properties from client failed %v\n", err)
			requestError(w, http.StatusInternalServerError)
			return
		}

		u, _ := url.Parse("https://github.com")
		if protocol, ok := requestedRepo["protocol"]; ok {
			u.Scheme = protocol
		}
		if host, ok := requestedRepo["host"]; ok {
			u.Host = host
		}
		if path, ok := requestedRepo["path"]; ok {
			u.Path = path
		}

		tokenResponse, err := vendor(r.Context(), *claims, u.String())
		if err != nil {
			log.Info().Msgf("token creation failed %v\n", err)
			requestError(w, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")

		// repo mismatch: empty return
		if tokenResponse == nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Length", "0")

			return
		}

		// write the reponse to the client in git credentials property format
		tokenURL, err := tokenResponse.URL()
		if err != nil {
			log.Info().Msgf("invalid repo URL: %v\n", err)
			requestError(w, http.StatusInternalServerError)
			return
		}

		err = credentialhandler.WriteProperties(map[string]string{
			"protocol":            tokenURL.Scheme,
			"host":                tokenURL.Host,
			"path":                tokenURL.Path,
			"username":            "x-access-token",
			"password":            tokenResponse.Token,
			"password_expiry_utc": tokenResponse.ExpiryUnix(),
		}, w)
		if err != nil {
			log.Info().Msgf("failed to write response: %v\n", err)
			requestError(w, http.StatusInternalServerError)
			return
		}
	})
}

func requestError(w http.ResponseWriter, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

type PipelineRepositoryToken struct {
	OrganizationSlug string    `json:"organizationSlug"`
	PipelineSlug     string    `json:"pipelineSlug"`
	RepositoryURL    string    `json:"repositoryUrl"`
	Token            string    `json:"token"`
	Expiry           time.Time `json:"expiry"`
}

func (t PipelineRepositoryToken) URL() (*url.URL, error) {
	url, err := url.Parse(t.RepositoryURL)
	if err != nil {
		return nil, err
	}

	return url, nil
}

func (t PipelineRepositoryToken) ExpiryUnix() string {
	return strconv.FormatInt(t.Expiry.UTC().Unix(), 10)
}

func IssueTokenForPipeline(
	repoLookup RepositoryLookup,
	tokenVendor TokenVendor,
) PipelineTokenVendor {
	return func(ctx context.Context, claims jwt.BuildkiteClaims, requestedRepoURL string) (*PipelineRepositoryToken, error) {
		// use buildkite api to find the repository for the pipeline
		pipelineRepoURL, err := repoLookup(ctx, claims.OrganizationSlug, claims.PipelineSlug)
		if err != nil {
			return nil, fmt.Errorf("could not find repository for pipeline %s: %w", claims.PipelineSlug, err)
		}

		if requestedRepoURL != "" && pipelineRepoURL != requestedRepoURL {
			// git is asking for a different repo than we can handle: return nil
			// to indicate that the handler should return a successful (but
			// empty) response.
			log.Info().Msgf("no token issued: repo mismatch. pipeline(%s) != requested(%s)\n", pipelineRepoURL, requestedRepoURL)
			return nil, nil
		}

		// use the github api to vend a token for the repository
		token, expiry, err := tokenVendor(ctx, pipelineRepoURL)
		if err != nil {
			return nil, fmt.Errorf("could not issue token for repository %s: %w", pipelineRepoURL, err)
		}

		log.Info().
			Str("organization", claims.OrganizationSlug).
			Str("pipeline", claims.PipelineSlug).
			Str("repo", requestedRepoURL).
			Msg("token issued")

		return &PipelineRepositoryToken{
			OrganizationSlug: claims.OrganizationSlug,
			PipelineSlug:     claims.PipelineSlug,
			RepositoryURL:    pipelineRepoURL,
			Token:            token,
			Expiry:           expiry,
		}, nil
	}
}

func newCachedPipelineTokenVendor() (func(PipelineTokenVendor) PipelineTokenVendor, error) {
	cache, err := otter.
		MustBuilder[string, PipelineRepositoryToken](10_000).
		CollectStats().
		WithTTL(45 * time.Minute).
		Build()
	if err != nil {
		return nil, err
	}

	return func(v PipelineTokenVendor) PipelineTokenVendor {
		return func(ctx context.Context, claims jwt.BuildkiteClaims, repo string) (*PipelineRepositoryToken, error) {
			// Cache by pipeline: this allows token reuse across multiple
			// builds. It's likely that this will change if rules allow for
			// builds to be given different permissions based on the branch or
			// tag.
			key := claims.PipelineID

			if response, ok := cache.Get(key); ok {
				if repo == "" || response.RepositoryURL == repo {
					log.Info().Time("expiry", response.Expiry).Str("key", key).Msg("hit: existing token found for pipeline")

					return &response, nil
				} else {
					// token invalid: remove from cache and fall through to reissue and re-cache
					log.Info().Str("key", key).Str("expected", repo).Str("actual", response.RepositoryURL).Msg("invalid: cached token issued for different repository")
					// the delete is required as "set" is not guaranteed to write to the cache
					cache.Delete(key)
				}
			}

			// TODO: if the cached token is about to expire, we could request a new token in the background

			// cache miss

			token, err := v(ctx, claims, repo)
			if err != nil {
				return nil, err
			}

			cache.Set(key, *token)

			return token, nil
		}
	}, nil
}
