package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/jamestelfer/ghauth/internal/buildkite"
	"github.com/jamestelfer/ghauth/internal/config"
	"github.com/jamestelfer/ghauth/internal/github"
	"github.com/jamestelfer/ghauth/internal/jwt"
	"github.com/jamestelfer/ghauth/internal/observe"
	"github.com/jamestelfer/ghauth/internal/vendor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/justinas/alice"
)

type AuthServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
	RegisterOnShutdown(f func())
}

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

	vendorCache, err := vendor.Cached()
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
	cfg, err := config.Load(context.Background())
	if err != nil {
		return fmt.Errorf("configuration load failed: %w", err)
	}

	handler, err := configureServerRoutes(cfg)
	if err != nil {
		return fmt.Errorf("server routing configuration failed: %w", err)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	err = serveHTTP(cfg.Server, cfg.Observe, server)
	if err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func serveHTTP(serverCfg config.ServerConfig, observeConfig config.ObserveConfig, server AuthServer) error {
	serverCtx := context.Background()

	// capture shutdown signals to allow for graceful shutdown
	ctx, stop := signal.NotifyContext(serverCtx,
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	shutdownTelemetry, err := observe.Configure(serverCtx, observeConfig)
	if err != nil {
		return fmt.Errorf("telemetry bootstrap failed: %w", err)
	}
	server.RegisterOnShutdown(func() {
		log.Info().Msg("telemetry: shutting down")
		shutdownTelemetry(serverCtx)
		log.Info().Msg("telemetry: shutdown complete")
	})

	// Start the server in a new goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Int("port", serverCfg.Port).Msg("starting server")
		serverErr <- server.ListenAndServe()
	}()

	var startupError error

	select {
	case err := <-serverErr:
		// Error when starting HTTP server.
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("failed to start server")
		}
		// save this error to return, keep processing shutdown sequence
		startupError = err
	case <-ctx.Done():
		log.Info().Msg("server shutdown requested")
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// Gracefully stop the server, allowing a configurable amount of time for
	// in-flight requests to complete
	shutdownTimeout := time.Duration(serverCfg.ShutdownTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info().Msg("server shutdown complete")

	// if startup failed the error is returned
	return startupError
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
