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

	vendor := IssueTokenForPipeline(bk.RepositoryLookup, gh.CreateAccessToken)

	http.Handle("POST /token", authorized.Then(handlePostToken(vendor)))
	http.Handle("POST /git-credentials", authorized.Then(handlePostGitCredentials(vendor)))

	return nil
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

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
	// capture signals to gracefully shutdown the server
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)

	server := &http.Server{Addr: fmt.Sprintf(":%d", serverCfg.Port)}

	// Start the server in a new goroutine
	var serverErr error
	go func() {
		log.Info().Int("port", serverCfg.Port).Msg("starting server")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("failed to start server")
			serverErr = err

			// signal the main goroutine to exit gracefully
			signalChan <- syscall.SIGINT
		}
	}()

	sig := <-signalChan
	log.Info().Stringer("signal", sig).Msg("server shutdown requested")

	// Gracefully stop the server, allow up to 25 seconds for in-flight requests to complete
	// TODO config timeout
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer func() {
		// additional shutdown handling if required
		cancel()
	}()

	err := server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// if shutdown is successful but startup failed, the process should exit
	// with an error
	return serverErr
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
