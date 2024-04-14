package main

import (
	"context"
	"fmt"
	"os"
	"time"

	localjwt "github.com/jamestelfer/ghauth/internal/jwt"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/go-jose/go-jose.v2"
	"gopkg.in/go-jose/go-jose.v2/json"
	"gopkg.in/go-jose/go-jose.v2/jwt"
)

type Config struct {
	Audience         string `env:"UTIL_AUDIENCE, default=test-audience"`
	Subject          string `env:"UTIL_SUBJECT, default=test-subject"`
	Issuer           string `env:"UTIL_ISSUER, default=https://local.testing"`
	OrganizationSlug string `env:"UTIL_BUILDKITE_ORGANIZATION_SLUG, required"`
	PipelineSlug     string `env:"UTIL_BUILDKITE_PIPELINE_SLUG, required"`
}

func main() {
	cfg := Config{}
	err := envconfig.Process(context.Background(), &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}

	jwksPath := ".development/keys/jwks.private.json"

	jwksBytes, err := os.ReadFile(jwksPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading jwks: %v\n", err)
		os.Exit(1)
	}

	jwks := jose.JSONWebKeySet{}
	err = json.Unmarshal(jwksBytes, &jwks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading jwks: %v\n", err)
		os.Exit(1)
	}

	key := jwks.Key("test-key")[0]

	jwt, err := createJWT(&key, validity(jwt.Claims{
		Audience: []string{cfg.Audience},
		Subject:  cfg.Subject,
		Issuer:   cfg.Issuer,
	}), localjwt.BuildkiteClaims{
		OrganizationSlug: cfg.OrganizationSlug,
		PipelineSlug:     cfg.PipelineSlug,
		PipelineID:       cfg.PipelineSlug + "UUID",
		BuildNumber:      123,
		BuildBranch:      "main",
		BuildCommit:      "abc123",
		StepKey:          "step1",
		JobId:            "job1",
		AgentId:          "agent1",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating JWT: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s", jwt)
}

func createJWT(jwk *jose.JSONWebKey, claims ...any) (string, error) {
	key := jose.SigningKey{
		Algorithm: jose.SignatureAlgorithm(jwk.Algorithm),
		Key:       jwk,
	}

	signer, err := jose.NewSigner(
		key,
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}

	builder := jwt.Signed(signer)

	for _, claim := range claims {
		builder = builder.Claims(claim)
	}

	token, err := builder.CompactSerialize()
	if err != nil {
		return "", err
	}

	return token, nil
}

func validity(claims jwt.Claims) jwt.Claims {
	now := time.Now().UTC()

	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now.Add(-1 * time.Minute))
	claims.Expiry = jwt.NewNumericDate(now.Add(1 * time.Minute))

	return claims
}
