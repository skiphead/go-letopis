package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/skiphead/go-letopis/internal/initapp"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Initialize application
	app, err := initapp.Initialize(*configPath)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run application in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := app.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	// Wait for termination signal or error
	select {
	case <-sigChan:
		app.Logger.Info("Received shutdown signal")
		cancel()
	case err := <-errCh:
		app.Logger.Error("Application error", "error", err)
		cancel()
	}

	// Wait for graceful shutdown
	<-ctx.Done()
	app.Logger.Info("Application shutdown complete")
}
