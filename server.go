package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"net/http"
	"io/fs"
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
	// Convert embed.FS to http.FileSystem
	httpFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(httpFS))

	// Handle root path to serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Serve the embedded index.html file
			data, err := staticFiles.ReadFile("dist/index.html")
			if err != nil {
				http.Error(w, "Could not read index.html", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
		} else {
			// Serve other static files
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
		Handler:   http.DefaultServeMux,
		TLSConfig: tlsConfig,
	}

	fmt.Println("Starting HTTPS server on https://127.0.0.1:443")
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("HTTPS server failed to start: %v", err)
	}
}
