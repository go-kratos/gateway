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
	"google.golang.org/grpc/benchmark/stats"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

var (
	length   int
	thread   int
	conns    int
	duration time.Duration
	flagSet  = flag.NewFlagSet("bench", flag.ExitOnError)
)

var (
	name       string
	mu         sync.Mutex
	wg         sync.WaitGroup
	errorCount = make(map[string]int)
	hists      []*stats.Histogram
	hopts      = stats.HistogramOptions{
		NumBuckets:   2495,
		GrowthFactor: .01,
	}

	wrr      int64
	transfer int64
	success  int64
	failure  int64
)

func init() {
	flagSet.IntVar(&length, "b", 1, "Length of request message")
	flagSet.IntVar(&conns, "c", 1, "Connections to keep open")
	flagSet.IntVar(&thread, "t", 1, " Number of thread to use")
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
	fmt.Printf("- %d threads and %d connections\n", thread, conns)

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
	for i := 0; i < thread; i++ {
		go worker(ctx, target, clients)
		time.Sleep(duration / time.Duration(thread) / 2)
	}
	start := time.Now()
	time.Sleep(duration)
	cancel()
	wg.Wait()
	gap := time.Since(start)
	suc := atomic.LoadInt64(&success)
	fail := atomic.LoadInt64(&failure)
	fmt.Printf("Requests/sec: %d\n", int64(float64(suc+fail)/gap.Seconds()))
	fmt.Printf("Transfer/sec: %d\n", transfer)
	printHist()
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
		mu.Lock()
		errorCount[err.Error()]++
		mu.Unlock()
		atomic.AddInt64(&failure, 1)
		return
	}
	atomic.AddInt64(&success, 1)
	atomic.AddInt64(&transfer, int64(len(reply.Message)))
}

func worker(ctx context.Context, target string, clients []pb.GreeterClient) {
	n := int64(len(clients))
	hist := stats.NewHistogram(hopts)
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			hists = append(hists, hist)
			mu.Unlock()
			return
		default:
			start := time.Now()
			idx := atomic.AddInt64(&wrr, 1) % n
			do(clients[idx])
			elapsed := time.Since(start)
			hist.Add(elapsed.Nanoseconds())
		}
	}
}

func printHist() {
	hist := stats.NewHistogram(hopts)
	for _, h := range hists {
		hist.Merge(h)
	}
	fmt.Printf("Latency: (50/90/99 %%ile): %v/%v/%v\n",
		time.Duration(median(.5, hist)),
		time.Duration(median(.9, hist)),
		time.Duration(median(.99, hist)))
}

func median(percentile float64, h *stats.Histogram) int64 {
	need := int64(float64(h.Count) * percentile)
	have := int64(0)
	for _, bucket := range h.Buckets {
		count := bucket.Count
		if have+count >= need {
			percent := float64(need-have) / float64(count)
			return int64((1.0-percent)*bucket.LowBound + percent*bucket.LowBound*(1.0+hopts.GrowthFactor))
		}
		have += bucket.Count
	}
	panic("should have found a bound")
}
