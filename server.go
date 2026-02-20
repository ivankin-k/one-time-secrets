package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Secret struct {
	ID       string    `json:"id"`
	Data     string    `json:"data"`
	ExpireAt time.Time `json:"expireAt"`
}

var (
	secretsStore *os.Root
)

func generateId() (id string) {
	var (
		source rand.Source
	)

	for range idLength {
		source = rand.NewSource(time.Now().UnixNano())
		id += string(idUsedSymbols[rand.New(source).Intn(len(idUsedSymbols))])
	}

	return id
}

func getSecret(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		secretId   string
		match      bool
		secretFile *os.File
		buffer     bytes.Buffer
		secret     Secret
	)

	// Check if valid secretId is received
	secretId = r.PathValue("secretId")
	match, err = regexp.MatchString("^[a-z0-9]{"+strconv.Itoa(idLength)+"}$", secretId)
	if err != nil || match == false {
		http.NotFound(w, r)
		return
	}

	// Check if secret exists
	_, err = secretsStore.Stat(secretId)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Open secret file and defer closing
	secretFile, err = secretsStore.OpenFile(secretId, os.O_RDONLY, 0600)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer secretFile.Close()

	// Read from file to buffer
	_, err = buffer.ReadFrom(secretFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Deserialize JSON
	secret = Secret{}
	err = json.Unmarshal(buffer.Bytes(), &secret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Defer a file removal (after deferred Close())
	defer secretsStore.Remove(secretId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buffer.Bytes())
}

func postSecret(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		buffer     bytes.Buffer
		secret     Secret
		secretFile *os.File
		data       []byte
		ip         string
	)

	// Check if IP rate limit is not violated
	ip, _, err = net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isCached(ip) {
		http.Error(w, fmt.Sprintf("Requests more frequently than once every %d seconds are not allowed.", ipDelaySec),
			http.StatusTooManyRequests)
		return
	}

	// Check if correct Content-Type is received
	if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		http.Error(w, "Incorrect \"Content-Type\"", http.StatusBadRequest)
		return
	}

	// Read request body and defer close
	_, err = buffer.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Deserialize request JSON data
	secret = Secret{}
	err = json.Unmarshal(buffer.Bytes(), &secret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if Data is received
	if secret.Data == "" {
		http.Error(w, "Data to save secretly is missing", http.StatusBadRequest)
		return
	}

	// Generate a unique ID
	for {
		secret.ID = generateId()
		_, err = secretsStore.Stat(secret.ID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Open file
	secretFile, err = secretsStore.OpenFile(secret.ID, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer secretFile.Close()

	// Serialize
	data, err = json.Marshal(secret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write to file
	_, err = secretFile.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update IP cache
	setCache(ip)

	// Send generated secret info
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://" + host + ":" + port + "/" + secret.ID))
}

func main() {
	var (
		err    error
		router *http.ServeMux
		info   os.FileInfo
	)

	// Create a store path (if needed) and get it's info
	for info == nil {
		info, err = os.Stat(storePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = os.Mkdir(storePath, 0700)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal(err)
			}
		}
	}

	// Check it's a directory
	if !info.IsDir() {
		log.Fatalf("Not a directory: %s", storePath)
	}

	// Set up folder root (secrets store) to work with
	secretsStore, err = os.OpenRoot(storePath)
	if err != nil {
		log.Fatal(err)
	}
	defer secretsStore.Close()

	// Set up routing and start the server
	router = http.NewServeMux()

	router.HandleFunc("GET /{secretId}", getSecret)
	router.HandleFunc("POST /{$}", postSecret)

	log.Print("Starting the server at " + host + ":" + port + "...")
	err = http.ListenAndServe(host+":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}
