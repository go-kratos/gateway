package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/helloworld/foo", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "bar")
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}
