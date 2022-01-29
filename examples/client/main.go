package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

var (
	length     int
	concurrent int
	conns      int
	duration   time.Duration
	flagSet    = flag.NewFlagSet("bench", flag.ExitOnError)
)

var (
	name       string
	lk         sync.Mutex
	errorCount = make(map[string]int)

	wrr      int64
	transfer int64
	success  int64
	failure  int64
)

func init() {
	flagSet.IntVar(&length, "b", 1, "Length of request message")
	flagSet.IntVar(&conns, "c", 1, "Connections to keep open")
	flagSet.IntVar(&concurrent, "t", 1, " Number of concurrent to use")
	flagSet.DurationVar(&duration, "d", time.Second*30, "Duration of test")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("client target is required")
		return
	}
	target := os.Args[1]
	flagSet.Parse(os.Args[2:])
	fmt.Printf("Running %v test @ %s\n", duration, target)
	for i := 0; i < length; i++ {
		name += "1"
	}
	var (
		ctx, cancel = context.WithCancel(context.Background())
		clients     []pb.GreeterClient
	)
	for i := 0; i < conns; i++ {
		conn, err := grpc.Dial(target, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		clients = append(clients, pb.NewGreeterClient(conn))
	}
	for i := 0; i < concurrent; i++ {
		go worker(ctx, target, clients)
	}
	start := time.Now()
	time.Sleep(duration)
	cancel()
	gap := time.Since(start)
	suc := atomic.LoadInt64(&success)
	fail := atomic.LoadInt64(&failure)
	fmt.Printf("Requests/sec: %d\n", int64(float64(suc+fail)/gap.Seconds()))
	fmt.Printf("Transfer/sec: %d\n", transfer)
	if len(errorCount) > 0 {
		fmt.Printf("Failures: %d\n", fail)
		for k, v := range errorCount {
			fmt.Printf("- %s : %d\n", k, v)
		}
	}
}

func do(client pb.GreeterClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	reply, err := client.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		lk.Lock()
		errorCount[err.Error()]++
		lk.Unlock()
		atomic.AddInt64(&failure, 1)
		return
	}
	atomic.AddInt64(&success, 1)
	atomic.AddInt64(&transfer, int64(len(reply.Message)))
}

func worker(ctx context.Context, target string, clients []pb.GreeterClient) {
	n := int64(len(clients))
	for {
		select {
		case <-ctx.Done():
			return
		default:
			idx := atomic.AddInt64(&wrr, 1) % n
			do(clients[idx])
		}
	}
}
