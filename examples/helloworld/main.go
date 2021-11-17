package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
)

var bind string

func init() {
	flag.StringVar(&bind, "bind", ":8000", "server address, eg: 127.0.0.1:8000")
}

func main() {
	flag.Parse()
	http.HandleFunc("/helloworld/foo", func(w http.ResponseWriter, req *http.Request) {
		b := req.URL.Query().Get("b")
		n, _ := strconv.ParseInt(b, 10, 32)
		w.Write(make([]byte, n))
	})
	log.Println("listening on:", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
