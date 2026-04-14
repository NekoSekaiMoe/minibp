// Package main is the entry point for the minibp server application.
// This file demonstrates how to use the api package and displays version information.

// main.go - Server entry point
package main

import (
	"fmt"

	// Import the API package from the local examples/src directory
	api "minibp/examples/src"
)

// main is the application entry point.
// It initializes the server and displays version information.
func main() {
	// Print the API version to the console
	// This demonstrates accessing the exported Version constant from the api package
	fmt.Printf("API Version: %s\n", api.Version)
}
