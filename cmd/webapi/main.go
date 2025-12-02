package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haembene/wasatext/service/api"
	"github.com/haembene/wasatext/service/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	//  LOGGER
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting WASAText server")

	//  DATABASE
	logger.Info("Opening SQLite database")

	dbconn, err := sql.Open("sqlite3", "./wasatext.db")
	if err != nil {
		return fmt.Errorf("opening sqlite db: %w", err)
	}
	defer func() { _ = dbconn.Close() }()

	db, err := database.New(dbconn)
	if err != nil {
		return fmt.Errorf("creating AppDatabase: %w", err)
	}

	//  API ROUTER
	logger.Info("Initializing API router")

	apirouter, err := api.New(api.Config{
		Logger:   logger,
		Database: db,
	})
	if err != nil {
		return fmt.Errorf("creating API router: %w", err)
	}

	router := apirouter.Handler()

	// HTTP SERVER
	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Infof("API is listening on http://localhost:8080")
		serverErrors <- server.ListenAndServe()
	}()

	// SHUTDOWN HANDLING
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Infof("Received signal %v, shutting down...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := apirouter.Close(); err != nil {
			logger.WithError(err).Warning("graceful shutdown error")
		}
		err := server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	logger.Info("Server stopped gracefully")
	return nil
}
