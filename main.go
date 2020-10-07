package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const ForwarderPrefix = "FORWARDER_"

var excludeExenions []string
var replaces map[string]string
var Version string

// Get the env variables required for a reverse proxy
func init() {
	fmt.Printf("forwarder %s\n", Version)
	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Proxy backend: %s\n", getProxyBackend())
	replaces = getReplace()
	log.Printf("Replace: %v\n", replaces)
	exes := getExcludeExtensions()
	if len(exes) > 0 {
		excludeExenions = exes
	} else {
		excludeExenions = []string{"jpg", "js", "css", "png", "webp", "jpeg"}
	}
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

func getReplace() map[string]string {
	out := map[string]string{}
	splits := strings.Split(getEnv("REPLACE", ""), ",")
	for _, split := range splits {
		inSplits := strings.Split(split, "=")
		if len(inSplits) == 2 {
			out[inSplits[0]] = inSplits[1]
		}
	}
	return out
}

func getExcludeExtensions() []string {
	return strings.Split(getEnv("EXCLUDE_EXTENSIONS", ""), ",")
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
	proxy.Transport = &transport{http.DefaultTransport}

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

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		defer reader.Close()
		resp.Header.Del("Content-Encoding")
	default:
		reader = resp.Body
	}

	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	for replace, value := range replaces {
		b = bytes.ReplaceAll(b, []byte(replace), []byte(value))
	}

	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	resp.ContentLength = int64(len(b))
	resp.Header.Set("Content-Length", strconv.Itoa(len(b)))

	return resp, nil
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
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
		http.HandleFunc("/ping", pingHandler)
		http.HandleFunc("/", handleRequestAndRedirect)
		if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
			panic(err)
		}
	}
}
