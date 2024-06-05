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
	muxWithoutTelemetry := http.NewServeMux()
	mux := observe.NewMux(muxWithoutTelemetry)

	// configure middleware
	authorizer, err := jwt.Middleware(cfg.Authorization, jwtmiddleware.WithErrorHandler(jwt.LogErrorHandler()))
	if err != nil {
		return nil, fmt.Errorf("authorizer configuration failed: %w", err)
	}

	// The request body size is fairly limited to prevent accidental or
	// deliberate abuse. Given the current API shape, this is not configurable.
	requestLimitBytes := int64(20 << 10) // 20 KB

	authorized := alice.New(maxRequestSize(requestLimitBytes), authorizer)

	// setup token handler and dependencies
	bk, err := buildkite.New(cfg.Buildkite)
	if err != nil {
		return nil, fmt.Errorf("buildkite configuration failed: %w", err)
	}

	gh, err := github.New(cfg.Github)
	if err != nil {
		return nil, fmt.Errorf("github configuration failed: %w", err)
	}

	vendorCache, err := vendor.Cached(45 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("vendor cache configuration failed: %w", err)
	}

	permissions, err := github.GetPermissionsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("getting permissions from environment variables failed: %v\n", err)
	}

	createAccessToken := func(ctx context.Context, repoURL string) (string, time.Time, error) {
		return gh.CreateAccessToken(ctx, repoURL, permissions)
	}

	tokenVendor := vendorCache(vendor.New(bk.RepositoryLookup, createAccessToken))

	mux.Handle("POST /token", authorized.Then(handlePostToken(tokenVendor)))
	mux.Handle("POST /git-credentials", authorized.Then(handlePostGitCredentials(tokenVendor)))

	// healthchecks are not included in telemetry
	muxWithoutTelemetry.Handle("GET /healthcheck", handleHealthCheck())

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

	// configure telemetry, including wrapping default HTTP client
	shutdownTelemetry, err := observe.Configure(ctx, cfg.Observe)
	if err != nil {
		return fmt.Errorf("telemetry bootstrap failed: %w", err)
	}

	http.DefaultTransport = observe.HttpTransport(
		configureHttpTransport(cfg.Server),
		cfg.Observe,
	)
	http.DefaultClient = &http.Client{
		Transport: http.DefaultTransport,
	}

	// setup routing and dependencies
	handler, err := configureServerRoutes(cfg)
	if err != nil {
		return fmt.Errorf("server routing configuration failed: %w", err)
	}

	// start the server
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        handler,
		MaxHeaderBytes: 20 << 10, // 20 KB
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
	// Set global level to the minimum: allows the Open Telemetry logging to be
	// configured separately. However, it means that any logger that sets its
	// level will log as this effectively disables the global level.
	zerolog.SetGlobalLevel(zerolog.Level(-128))

	// default level is Info
	log.Logger = log.Level(zerolog.InfoLevel)

	if os.Getenv("ENV") == "development" {
		log.Logger = log.
			Output(zerolog.ConsoleWriter{Out: os.Stdout}).
			Level(zerolog.DebugLevel)
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

func configureHttpTransport(cfg config.ServerConfig) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.MaxIdleConns = cfg.OutgoingHttpMaxIdleConns
	transport.MaxConnsPerHost = cfg.OutgoingHttpMaxConnsPerHost

	return transport
}
