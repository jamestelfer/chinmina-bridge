package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func configureServerRoutes() {
	http.HandleFunc("POST /token", handlePostToken)
}

func main() {
	configureServerRoutes()

	err := serveHTTP()
	if err != nil {
		fmt.Printf("server failed: %v\n", err)
		os.Exit(1)
	}
}

func serveHTTP() error {
	// Setup signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)

	// TODO: config port
	server := &http.Server{Addr: ":8080"}

	// Start the server in a new goroutine
	var serverErr error
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
			serverErr = err

			// signal the main goroutine to exit gracefully
			signalChan <- syscall.SIGINT
		}
	}()

	sig := <-signalChan
	fmt.Printf("Received server shutdown signal: %v\n", sig)

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
