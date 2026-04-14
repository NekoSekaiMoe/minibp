// Package api provides types for the minibp API service.
// This file contains the main service implementation and API version information.

// api.go - Go API package
package api

import "fmt"

// Version defines the current API version string.
// This follows semantic versioning and is used for compatibility checking
// and informational purposes.
const Version = "1.0.0"

// Service represents an API service instance.
// It provides lifecycle methods to start and stop the service.
type Service struct {
	// Name is the identifier for this service instance.
	// It is used for logging and identification purposes.
	Name string
}

// Start initializes and starts the service.
// It performs any necessary initialization tasks such as opening connections,
// setting up handlers, and beginning to accept requests.
//
// Returns nil if the service started successfully, or an error if
// initialization failed.
func (s *Service) Start() error {
	fmt.Printf("Starting service: %s\n", s.Name)
	return nil
}

// Stop gracefully shuts down the service.
// It performs cleanup tasks such as closing connections, flushing buffers,
// and releasing resources.
//
// Returns nil if the service stopped cleanly, or an error if
// cleanup encountered problems.
func (s *Service) Stop() error {
	fmt.Printf("Stopping service: %s\n", s.Name)
	return nil
}
