package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var (
	addr   string
	target string
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "server address, eg: 127.0.0.1:8080")
	flag.StringVar(&target, "target", "http://127.0.0.1:8000", "proxy target, eg: http://127.0.0.1:8000")
}

func newProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}
	rp, err := httputil.NewSingleHostReverseProxy(url), nil
	if err != nil {
		return nil, err
	}
	rp.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   200 * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		DisableCompression:    true,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return rp, nil
}

func proxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

func main() {
	flag.Parse()
	proxy, err := newProxy(target)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", proxyRequestHandler(proxy))
	log.Println("server listening on:", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
