package nanotime

import (
	"time"
	_ "unsafe"
)

//go:linkname RuntimeNanotime runtime.nanotime
func RuntimeNanotime() int64

func SinceSeconds(t int64) float64 {
	return Since(t, time.Second)
}

func Since(t int64, unit time.Duration) float64 {
	return float64(RuntimeNanotime()-t) / float64(unit)
}
