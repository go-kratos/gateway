package main

import (
	"flag"
	"io"
	"log"
	"net/http"
)

var bind string

func init() {
	flag.StringVar(&bind, "bind", ":8000", "server address, eg: 127.0.0.1:8000")
}

func main() {
	flag.Parse()
	http.HandleFunc("/helloworld/foo", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "bar")
	})
	log.Println("listening on:", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
