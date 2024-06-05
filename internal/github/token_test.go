package github_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/google/go-github/v61/github"
	"github.com/jamestelfer/chinmina-bridge/internal/config"
	"github.com/jamestelfer/chinmina-bridge/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccessToken_Succeeds(t *testing.T) {
	router := http.NewServeMux()

	expectedExpiry := time.Date(1980, 01, 01, 0, 0, 0, 0, time.UTC)
	actualInstallation := "unknown"
	var actualPermissions map[string]string

	router.HandleFunc("/app/installations/{installationID}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		actualInstallation = r.PathValue("installationID")

		var reqBody struct {
			Permissions map[string]string `json:"permissions"`
		}
		json.NewDecoder(r.Body).Decode(&reqBody)
		actualPermissions = reqBody.Permissions

		JSON(w, &api.InstallationToken{
			Token:     api.String("expected-token"),
			ExpiresAt: &api.Timestamp{Time: expectedExpiry},
		})
	})

	svr := httptest.NewServer(router)
	defer svr.Close()

	// generate valid key for testing
	key := generateKey(t)

	gh, err := github.New(config.GithubConfig{
		ApiURL:         svr.URL,
		PrivateKey:     key,
		ApplicationID:  10,
		InstallationID: 20,
	})
	require.NoError(t, err)

	permissions := github.Permissions{
		Contents:     "read",
		PullRequests: "read",
	}

	token, expiry, err := gh.CreateAccessToken(context.Background(), "https://github.com/organization/repository", permissions)

	require.NoError(t, err)
	assert.Equal(t, "expected-token", token)
	assert.Equal(t, expectedExpiry, expiry)
	assert.Equal(t, "20", actualInstallation)
	assert.Equal(t, "read", actualPermissions["contents"])
	assert.Equal(t, "read", actualPermissions["pull_requests"])
}

func TestCreateAccessToken_Fails_On_Invalid_URL(t *testing.T) {
	router := http.NewServeMux()

	router.HandleFunc("/app/installations/{installationID}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	svr := httptest.NewServer(router)
	defer svr.Close()

	// generate valid key for testing
	key := generateKey(t)

	gh, err := github.New(config.GithubConfig{
		ApiURL:         svr.URL,
		PrivateKey:     key,
		ApplicationID:  10,
		InstallationID: 20,
	})
	require.NoError(t, err)

	permissions := github.Permissions{
		Contents:     "read",
		PullRequests: "read",
	}

	_, _, err = gh.CreateAccessToken(context.Background(), "sch_eme://invalid_url/", permissions)

	require.Error(t, err)
	assert.ErrorContains(t, err, "first path segment in URL")
}

func TestCreateAccessToken_Fails_On_Failed_Request(t *testing.T) {
	router := http.NewServeMux()

	router.HandleFunc("/app/installations/{installationID}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	svr := httptest.NewServer(router)
	defer svr.Close()

	// generate valid key for testing
	key := generateKey(t)

	gh, err := github.New(config.GithubConfig{
		ApiURL:         svr.URL,
		PrivateKey:     key,
		ApplicationID:  10,
		InstallationID: 20,
	})
	require.NoError(t, err)

	permissions := github.Permissions{
		Contents:     "read",
		PullRequests: "read",
	}

	_, _, err = gh.CreateAccessToken(context.Background(), "https://dodgey", permissions)

	require.Error(t, err)
	assert.ErrorContains(t, err, ": 418")
}

func JSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	res, _ := json.Marshal(payload)
	_, _ = w.Write(res)
}

// generateKey creates and PEM encodes a valid RSA private key for testing.
func generateKey(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	key := pem.EncodeToMemory(privateKeyPEM)

	return string(key)
}
