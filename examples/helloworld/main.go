package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/helloworld/foo", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "bar")
	})
	srv := &http.Server{
		Addr: ":8000",
		Handler: h2c.NewHandler(mux, &http2.Server{
			IdleTimeout: time.Second * 120,
		}),
		ReadTimeout:       time.Second * 1,
		ReadHeaderTimeout: time.Second * 1,
		WriteTimeout:      time.Second * 1,
		IdleTimeout:       time.Second * 120,
	}
	log.Fatal(srv.ListenAndServe())
}
