package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/jamestelfer/ghauth/internal/credentialhandler"
	"github.com/jamestelfer/ghauth/internal/jwt"
	"github.com/jamestelfer/ghauth/internal/vendor"
	"github.com/rs/zerolog/log"
)

func handlePostToken(tokenVendor vendor.PipelineTokenVendor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// claims must be present from the middleware
		claims := jwt.RequireBuildkiteClaimsFromContext(r.Context())

		tokenResponse, err := tokenVendor(r.Context(), claims, "")
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

func handlePostGitCredentials(tokenVendor vendor.PipelineTokenVendor) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure that the request body is fully read prior to returning. This
		// avoids issues with blocked connections and connection reuse.
		defer func() { io.Copy(io.Discard, r.Body) }()

		// claims must be present from the middleware
		claims := jwt.RequireBuildkiteClaimsFromContext(r.Context())

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

		tokenResponse, err := tokenVendor(r.Context(), claims, u.String())
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
