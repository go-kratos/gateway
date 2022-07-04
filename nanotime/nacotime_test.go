package nanotime

import (
	"testing"
	"time"
)

func BenchmarkTimeNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cost := time.Now()
		time.Since(cost).Seconds()
	}
}

func BenchmarkRuntimeNanotime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cost := RuntimeNanotime()
		_ = SinceSeconds(cost)
	}
}

func TestProxy_Time(t *testing.T) {
	n1 := time.Now()
	n2 := RuntimeNanotime()
	time.Sleep(time.Second)
	t.Log(time.Since(n1).Seconds())
	t.Log(SinceSeconds(n2))
}
