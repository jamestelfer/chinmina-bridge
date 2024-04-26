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
	"github.com/jamestelfer/ghauth/internal/vendor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/justinas/alice"
)

func configureServerRoutes(cfg config.Config) error {
	// configure middleware
	authorizer, err := jwt.Middleware(cfg.Authorization, jwtmiddleware.WithErrorHandler(jwt.LogErrorHandler()))
	if err != nil {
		return fmt.Errorf("authorizer configuration failed: %w", err)
	}

	authorized := alice.New(authorizer)

	// setup token handler and dependencies
	bk := buildkite.New(cfg.Buildkite)
	gh, err := github.New(cfg.Github)
	if err != nil {
		return fmt.Errorf("github configuration failed: %w", err)
	}

	vendorCache, err := vendor.Cached()
	if err != nil {
		return fmt.Errorf("vendor cache configuration failed: %w", err)
	}

	tokenVendor := vendorCache(vendor.New(bk.RepositoryLookup, gh.CreateAccessToken))

	http.Handle("POST /token", authorized.Then(handlePostToken(tokenVendor)))
	http.Handle("POST /git-credentials", authorized.Then(handlePostGitCredentials(tokenVendor)))

	return nil
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

	err = configureServerRoutes(cfg)
	if err != nil {
		return fmt.Errorf("server routing configuration failed: %w", err)
	}

	err = serveHTTP(cfg.Server)
	if err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func serveHTTP(serverCfg config.ServerConfig) error {
	serverCtx := context.Background()

	// capture shutdown signals to allow for graceful shutdown
	ctx, stop := signal.NotifyContext(serverCtx,
		os.Interrupt, syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	server := &http.Server{Addr: fmt.Sprintf(":%d", serverCfg.Port)}

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
		startupError = err
	case <-ctx.Done():
		log.Info().Msg("server shutdown requested")
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// Gracefully stop the server, allow up to 25 seconds for in-flight requests to complete
	shutdownTimeout := time.Duration(serverCfg.ShutdownTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// if shutdown is successful but startup failed, the process should exit
	// with an error
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
