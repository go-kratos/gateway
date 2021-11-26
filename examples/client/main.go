package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

var (
	addr        string
	length      int
	concurrent  int
	duration    int
	connPerHost bool
)

var (
	name       string
	errorCount map[string]int
	lk         sync.Mutex

	success int64
	failure int64
)

func init() {
	flag.StringVar(&addr, "addr", "127.0.0.1:9000", "")
	flag.IntVar(&length, "length", 1, "")
	flag.IntVar(&duration, "duration", 3, "")
	flag.IntVar(&concurrent, "concurrent", 10, "")
	flag.BoolVar(&connPerHost, "connPerHost", false, "")

	errorCount = make(map[string]int)
}

func main() {
	flag.Parse()
	fmt.Println(addr, length, concurrent, duration, connPerHost)
	for i := 0; i < length; i++ {
		name += "1"
	}
	var client pb.GreeterClient
	if !connPerHost {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		client = pb.NewGreeterClient(conn)
	}
	for i := 0; i < concurrent; i++ {
		go worker(client)
	}
	start := time.Now()
	time.Sleep(time.Second * time.Duration(duration))
	gap := time.Since(start)
	suc := atomic.LoadInt64(&success)
	fail := atomic.LoadInt64(&failure)
	fmt.Println("gap:", gap)
	fmt.Println("qps:", int64(float64(suc+fail)/gap.Seconds()))
	fmt.Println("failure:", fail)
	lk.Lock()
	fmt.Println(errorCount)
	lk.Unlock()
}

func do(client pb.GreeterClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	reply, err := client.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil || reply.Message == "" {
		atomic.AddInt64(&failure, 1)
		if err != nil {
			lk.Lock()
			errorCount[err.Error()]++
			lk.Unlock()
		}
		return
	}
	atomic.AddInt64(&success, 1)
}

func worker(client pb.GreeterClient) {
	if client == nil {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		client = pb.NewGreeterClient(conn)
	}
	for {
		do(client)
	}
}
