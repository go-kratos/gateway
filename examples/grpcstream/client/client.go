package main

import (
	"context"
	"flag"
	"io"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/features/proto/echo"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "server address")
	input := flag.String("messages", "hello,world,stream", "comma-separated payloads to send")
	timeout := flag.Duration("timeout", 5*time.Second, "overall RPC deadline")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, *addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEchoClient(conn)
	stream, err := client.BidirectionalStreamingEcho(ctx)
	if err != nil {
		log.Fatalf("failed to start stream: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatalf("failed to receive: %v", err)
			}
			log.Printf("got response: %q", resp.GetMessage())
		}
	}()

	for _, msg := range strings.Split(*input, ",") {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		if err := stream.Send(&pb.EchoRequest{Message: msg}); err != nil {
			log.Fatalf("failed to send message %q: %v", msg, err)
		}
		time.Sleep(150 * time.Millisecond)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("failed to close send: %v", err)
	}

	<-done
	log.Printf("stream completed")
}
