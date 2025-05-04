package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"net/http"
	"io/fs"
	"os"
)

//go:embed dist/*
var staticFiles embed.FS

//go:embed certs/server.crt
var serverCert []byte

//go:embed certs/server.key
var serverKey []byte

// 将 httpsPort 声明为全局变量
var httpsPort string


// func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
// 	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
// }

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	// 确保在重定向时包含正确的 HTTPS 端口
	httpsHost := r.Host
	if httpsPort != "443" { // 如果 HTTPS 端口不是默认的 443
		httpsHost = fmt.Sprintf("%s:%s", r.Host, httpsPort)
	}
	http.Redirect(w, r, "https://"+httpsHost+r.RequestURI, http.StatusMovedPermanently)
}


func main() {
	// Get ports from environment variables, with default values
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "80"
	}

	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == "" {
		httpsPort = "443"
	}

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
		fmt.Printf("Starting HTTP server on http://127.0.0.1:%s\n", httpPort)
		if err := http.ListenAndServe(":"+httpPort, http.HandlerFunc(redirectToHTTPS)); err != nil {
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
		Addr:      ":" + httpsPort,
		Handler:   http.DefaultServeMux,
		TLSConfig: tlsConfig,
	}

	fmt.Printf("Starting HTTPS server on https://127.0.0.1:%s\n", httpsPort)
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("HTTPS server failed to start: %v", err)
	}
}
