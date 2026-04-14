// Package api provides types for the minibp API service.
// This file defines the core data structures used throughout the application.

// types.go - Go types
package api

// Config holds the server configuration settings.
// It is used to specify the network address and port where the server will listen.
type Config struct {
	// Host is the network interface or IP address the server will bind to.
	// Empty string or "0.0.0.0" means binding to all available interfaces.
	Host string
	// Port is the TCP port number the server will listen on.
	// Valid range is typically 1-65535.
	Port int
}

// Response represents the result of an API operation.
// It encapsulates either successful data or an error condition.
type Response struct {
	// Data contains the payload from a successful operation.
	// This field is populated when Error is nil.
	Data string
	// Error holds any error that occurred during processing.
	// This field is nil when the operation succeeded.
	Error error
}
