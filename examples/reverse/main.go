package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func newProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(url), nil
}

func proxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

func main() {
	proxy, err := newProxy("http://127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", proxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
