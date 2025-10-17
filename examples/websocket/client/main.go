package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/websocket"
)

var (
	serverAddr = flag.String("server", "ws://localhost:18080/ws", "websocket server url")
	message    = flag.String("message", "hello gateway", "text payload to send")
	count      = flag.Int("count", 1, "number of messages to send")
	interval   = flag.Duration("interval", time.Second, "pause between messages")
)

func main() {
	flag.Parse()

	u, err := url.Parse(*serverAddr)
	if err != nil {
		log.Fatalf("invalid server url: %v", err)
	}

	origin := &url.URL{Scheme: "http", Host: u.Host}
	log.Printf("connecting to %s", u.String())
	conn, err := websocket.Dial(u.String(), "", origin.String())
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.PayloadType = websocket.TextFrame

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err == nil {
		var welcome string
		if err := websocket.Message.Receive(conn, &welcome); err == nil {
			log.Printf("server greeting: %s", welcome)
		} else {
			log.Printf("waiting for welcome message failed: %v", err)
		}
		_ = conn.SetDeadline(time.Time{})
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	go func() {
		for {
			var payload string
			if err := websocket.Message.Receive(conn, &payload); err != nil {
				log.Printf("read error: %v", err)
				close(done)
				return
			}
			log.Printf("received: %s", payload)
		}
	}()

	for i := 0; i < *count; i++ {
		payload := fmt.Sprintf("%s (%d/%d)", *message, i+1, *count)
		if err := websocket.Message.Send(conn, payload); err != nil {
			log.Fatalf("write failed: %v", err)
		}
		time.Sleep(*interval)
	}

	select {
	case <-done:
		log.Printf("connection closed by server")
	case <-interrupt:
		log.Printf("interrupt received, closing connection")
	case <-time.After(2 * time.Second):
		log.Printf("no more messages, closing connection")
	}
}
