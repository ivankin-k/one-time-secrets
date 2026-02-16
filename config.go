// Configuration file

package main

const (
	HOST         = "127.0.0.1"        // Host to listen
	PORT         = "8000"             // Port to listen
	STORE_PATH   = "secrets"          // Name of folder to store secrets
	ID_LEN       = 32                 // Length of an ID
	ID_SYMBS     = "abcdef0123456789" // Symbols to be used in IDs
	IP_DELAY_SEC = 5                  // Wait (seconds) before another POST request
)
