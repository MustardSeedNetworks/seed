// Package main is the entry point for NetScope.
package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	devMode := flag.Bool("dev", false, "Run in development mode")
	port := flag.Int("port", 8443, "Server port")
	flag.Parse()

	if *showVersion {
		fmt.Printf("NetScope %s\n", version)
		os.Exit(0)
	}

	fmt.Printf("NetScope %s starting...\n", version)
	fmt.Printf("Mode: %s\n", map[bool]string{true: "development", false: "production"}[*devMode])
	fmt.Printf("Port: %d\n", *port)

	// TODO: Initialize application
	// - Load configuration
	// - Set up network interface monitoring
	// - Start packet capture
	// - Initialize WebSocket server
	// - Start HTTP server

	fmt.Println("NetScope is not yet implemented. See PROJECT_PLAN.md for roadmap.")
}
