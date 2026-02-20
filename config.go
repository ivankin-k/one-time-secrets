// Configuration file

package main

const (
	host          = "127.0.0.1"        // Host to listen
	port          = "8000"             // Port to listen
	storePath     = "secrets"          // Name of folder to store secrets
	idLength      = 32                 // Length of an ID
	idUsedSymbols = "abcdef0123456789" // Symbols to be used in IDs
	ipDelaySec    = 5                  // Wait (seconds) before another POST request
)
