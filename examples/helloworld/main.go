package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/go-kratos/kratos/v2/encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/protobuf/proto"
)

var (
	httpAddr string
	grpcAddr string
)

type codec struct{}

func (codec) Marshal(v interface{}) ([]byte, error) {
	vv, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to marshal, message is %T, want proto.Message", v)
	}
	return json.MarshalOptions.Marshal(vv)
}
func (codec) Unmarshal(data []byte, v interface{}) error {
	vv, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal, message is %T, want proto.Message", v)
	}
	return json.UnmarshalOptions.Unmarshal(data, vv)
}
func (codec) Name() string { return "json" }

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: in.GetName()}, nil
}

func init() {
	flag.StringVar(&httpAddr, "http.addr", ":8000", "server address, eg: 127.0.0.1:8000")
	flag.StringVar(&grpcAddr, "grpc.addr", ":9000", "server address, eg: 127.0.0.1:9000")
	encoding.RegisterCodec(codec{})
}

func main() {
	flag.Parse()
	go httpServer()
	grpcServer()
}

func httpServer() {
	http.HandleFunc("/helloworld/foo", func(w http.ResponseWriter, req *http.Request) {
		b := req.URL.Query().Get("b")
		if b != "" {
			n, _ := strconv.ParseInt(b, 10, 32)
			w.Write(make([]byte, n))
		}
	})
	log.Println("HTTPServer listening at:", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func grpcServer() {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	log.Println("GRPCServer listening at:", lis.Addr())
	log.Fatal(s.Serve(lis))
}
