package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/features/proto/echo"
)

type echoServer struct {
	pb.UnimplementedEchoServer
	artificialDelay time.Duration
}

func (s *echoServer) BidirectionalStreamingEcho(stream pb.Echo_BidirectionalStreamingEchoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream recv failed: %w", err)
		}
		log.Printf("received message: %q", req.GetMessage())

		if s.artificialDelay > 0 {
			time.Sleep(s.artificialDelay)
		}
		resp := &pb.EchoResponse{Message: fmt.Sprintf("S echo: %s", req.GetMessage())}
		if err := stream.Send(resp); err != nil {
			return fmt.Errorf("stream send failed: %w", err)
		}
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "listening address")
	delay := flag.Duration("delay", 200*time.Millisecond, "optional delay before each response")
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterEchoServer(server, &echoServer{artificialDelay: *delay})

	log.Printf("stream server listening on %s", *addr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
