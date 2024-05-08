package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/jamestelfer/chinmina-bridge/internal/buildkite"
	"github.com/jamestelfer/chinmina-bridge/internal/config"
	"github.com/jamestelfer/chinmina-bridge/internal/github"
	"github.com/jamestelfer/chinmina-bridge/internal/jwt"
	"github.com/jamestelfer/chinmina-bridge/internal/observe"
	"github.com/jamestelfer/chinmina-bridge/internal/vendor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/justinas/alice"
)

func configureServerRoutes(cfg config.Config) (http.Handler, error) {
	// wrap a mux such that HTTP telemetry is configured by default
	mux := observe.NewMux(http.NewServeMux())

	// configure middleware
	authorizer, err := jwt.Middleware(cfg.Authorization, jwtmiddleware.WithErrorHandler(jwt.LogErrorHandler()))
	if err != nil {
		return nil, fmt.Errorf("authorizer configuration failed: %w", err)
	}

	authorized := alice.New(authorizer)

	// setup token handler and dependencies
	bk := buildkite.New(cfg.Buildkite)
	gh, err := github.New(cfg.Github)
	if err != nil {
		return nil, fmt.Errorf("github configuration failed: %w", err)
	}

	vendorCache, err := vendor.Cached(45 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("vendor cache configuration failed: %w", err)
	}

	tokenVendor := vendorCache(vendor.New(bk.RepositoryLookup, gh.CreateAccessToken))

	mux.Handle("POST /token", authorized.Then(handlePostToken(tokenVendor)))
	mux.Handle("POST /git-credentials", authorized.Then(handlePostGitCredentials(tokenVendor)))

	return mux, nil
}

func main() {
	configureLogging()

	logBuildInfo()

	err := launchServer()
	if err != nil {
		log.Fatal().Err(err).Msg("server failed to start")
	}
}

func launchServer() error {
	ctx := context.Background()

	cfg, err := config.Load(context.Background())
	if err != nil {
		return fmt.Errorf("configuration load failed: %w", err)
	}

	handler, err := configureServerRoutes(cfg)
	if err != nil {
		return fmt.Errorf("server routing configuration failed: %w", err)
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        handler,
		MaxHeaderBytes: 20 << 10, // 20 KB
	}

	shutdownTelemetry, err := observe.Configure(ctx, cfg.Observe)
	if err != nil {
		return fmt.Errorf("telemetry bootstrap failed: %w", err)
	}
	server.RegisterOnShutdown(func() {
		log.Info().Msg("telemetry: shutting down")
		shutdownTelemetry(ctx)
		log.Info().Msg("telemetry: shutdown complete")
	})

	err = serveHTTP(cfg.Server, server)
	if err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func configureLogging() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func logBuildInfo() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	ev := log.Info()
	for _, v := range buildInfo.Settings {

		if strings.HasPrefix(v.Key, "vcs.") ||
			strings.HasPrefix(v.Key, "GO") ||
			v.Key == "CGO_ENABLED" {
			ev = ev.Str(v.Key, v.Value)
		}
	}

	ev.Msg("build information")
}
