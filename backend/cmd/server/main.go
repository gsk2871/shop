// Command server starts the one backend service for public and admin APIs.
package main

import "peptide-site/internal/pkg/bootstrap"

// Version is set at build time with -ldflags "-X main.Version=...".
var Version string

func main() { bootstrap.Run(Version, initializeServer) }
