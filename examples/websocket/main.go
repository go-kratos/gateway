package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/websocket"
)

var (
	listenAddr = flag.String("addr", ":18080", "http listen address")
	wsPath     = flag.String("path", "/ws", "websocket endpoint path")
)

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Gateway websocket example\nConnect with ws://%s%s\n", r.Host, *wsPath)
	})

	// Allow all origins for demonstration so the CLI client and browsers can connect.
	http.Handle(*wsPath, websocket.Server{
		Handshake: func(config *websocket.Config, req *http.Request) error {
			if origin := req.Header.Get("Origin"); origin != "" {
				if u, err := url.Parse(origin); err == nil {
					config.Origin = u
				}
			}
			return nil
		},
		Handler: websocket.Handler(handleWebsocket),
	})

	log.Printf("websocket server listening on %s at %s", *listenAddr, *wsPath)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}

func handleWebsocket(conn *websocket.Conn) {
	defer conn.Close()
	conn.PayloadType = websocket.TextFrame

	welcome := fmt.Sprintf("hello from gateway websocket server (%s)", time.Now().Format(time.RFC3339))
	if err := websocket.Message.Send(conn, welcome); err != nil {
		log.Printf("write welcome failed: %v", err)
		return
	}

	for {
		var payload string
		if err := websocket.Message.Receive(conn, &payload); err != nil {
			log.Printf("read error: %v", err)
			return
		}

		log.Printf("received message: %q", payload)
		response := fmt.Sprintf("[%s] S echo: %s", time.Now().Format(time.RFC3339), payload)
		if err := websocket.Message.Send(conn, response); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}
