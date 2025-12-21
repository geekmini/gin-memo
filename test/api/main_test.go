//go:build api

// Package api contains API integration tests for the gin-sample application.
// These tests run against real MongoDB, Redis, and MinIO instances using testcontainers.
//
// Run tests with:
//
//	go test -tags=api -v ./test/api/...
//
// Or using the task runner:
//
//	task test:api
package api

import (
	"context"
	"log"
	"os"
	"testing"

	"gin-sample/internal/validator"
	"gin-sample/test/api/testserver"
)

// testServer is the global test server instance shared across all tests.
var testServer *testserver.TestServer

// TestMain sets up the test server and runs all tests.
func TestMain(m *testing.M) {
	// Register custom validators
	validator.RegisterCustomValidators()

	ctx := context.Background()

	// Setup: Start all containers and wire dependencies
	log.Println("Starting test containers...")
	var err error
	testServer, err = testserver.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create test server: %v", err)
	}
	log.Println("Test containers started successfully")

	// Run all tests
	code := m.Run()

	// Teardown: Stop all containers
	log.Println("Stopping test containers...")
	testServer.Cleanup(ctx)
	log.Println("Test containers stopped")

	os.Exit(code)
}
