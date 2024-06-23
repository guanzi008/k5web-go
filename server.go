// main.go
package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"net/http"
	"strings"
)

//go:embed dist/*
var staticFiles embed.FS

//go:embed certs/server.crt
var serverCert []byte

//go:embed certs/server.key
var serverKey []byte

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func main() {
	// Handle static files
	httpFS := http.FS(staticFiles)
	fileServer := http.FileServer(http.FS(httpFS))

	// Handle root path to serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "dist/index.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	// Start HTTP server for redirecting to HTTPS
	go func() {
		fmt.Println("Starting HTTP server on http://127.0.0.1:80")
		if err := http.ListenAndServe(":80", http.HandlerFunc(redirectToHTTPS)); err != nil {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// Load embedded certificates
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		log.Fatalf("Failed to load embedded certificates: %v", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Start HTTPS server
	server := &http.Server{
		Addr:      ":443",
		Handler:   http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fileServer.ServeHTTP(w, r)
		}),
		TLSConfig: tlsConfig,
	}

	fmt.Println("Starting HTTPS server on https://127.0.0.1:443")
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("HTTPS server failed to start: %v", err)
	}
}
