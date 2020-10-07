package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

const ForwarderPrefix = "FORWARDER_"

var Version string

// Get the env variables required for a reverse proxy
func init() {
	fmt.Printf("forwarder %s\n", Version)
	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Proxy backend: %s\n", getProxyBackend())
}

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(ForwarderPrefix + key); ok {
		return value
	}
	return fallback
}

// Get the port to listen on
func getListenAddress() string {
	port := getEnv("PORT", "8888")
	return "0.0.0.0:" + port
}

// Get the backend host and port
func getProxyBackend() string {
	proxyBackend := getEnv("PROXY_BACKEND", "http://localhost:8080")
	return proxyBackend
}

/*
	Reverse Proxy Logic
*/

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	url := getProxyBackend()
	serveReverseProxy(url, res, req)
}

/*
	Entry
*/

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("forwarder %s\n", Version)
		default:
			fmt.Printf("forwarder %s\n", Version)
		}
	} else {
		// start server
		http.HandleFunc("/", handleRequestAndRedirect)
		if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
			panic(err)
		}
	}
}
