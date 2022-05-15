package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var addr string

func init() {
	flag.StringVar(&addr, "addr", ":8080", "server address, eg: 127.0.0.1:8080")
}

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
	log.Println("server listening on:", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
