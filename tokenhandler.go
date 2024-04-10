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
			fmt.Printf("token creation failed %v\n", err)
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
			fmt.Printf("read repository properties from client failed %v\n", err)
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
			fmt.Printf("token creation failed %v\n", err)
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
		err = credentialhandler.WriteProperties(map[string]string{
			"protocol":            tokenResponse.URL().Scheme,
			"host":                tokenResponse.URL().Host,
			"path":                tokenResponse.URL().Path,
			"username":            "x-access-token",
			"password":            tokenResponse.Token,
			"password_expiry_utc": tokenResponse.ExpiryUnix(),
		}, w)
		if err != nil {
			fmt.Printf("failed to write response: %v\n", err)
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
	url              *url.URL
}

func (t *PipelineRepositoryToken) URL() *url.URL {
	if t.url != nil {
		return t.url
	}

	url, err := url.Parse(t.RepositoryURL)
	if err != nil {
		panic(err) //FIXME keep your towel handy
	}

	t.url = url

	return url
}

func (t *PipelineRepositoryToken) ExpiryUnix() string {
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
			fmt.Printf("no token issued: repo mismatch. pipeline(%s) != requested(%s)\n", pipelineRepoURL, requestedRepoURL)
			return nil, nil
		}

		// use the github api to vend a token for the repository
		token, expiry, err := tokenVendor(ctx, pipelineRepoURL)
		if err != nil {
			return nil, fmt.Errorf("could not issue token for repository %s: %w", pipelineRepoURL, err)
		}

		return &PipelineRepositoryToken{
			OrganizationSlug: claims.OrganizationSlug,
			PipelineSlug:     claims.PipelineSlug,
			RepositoryURL:    pipelineRepoURL,
			Token:            token,
			Expiry:           expiry,
		}, nil
	}
}
