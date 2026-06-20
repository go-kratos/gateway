//go:build wasip1

package main

import "unsafe"

var buf [1024]byte

//go:wasmimport env get_uri
func getUri(ptr uint32, limit uint32) uint32

//go:wasmimport env block_request
func blockRequest()

func main() {
	ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))
	n := getUri(ptr, 1024)

	if n == 10 &&
		buf[0] == '/' && buf[1] == 'f' && buf[2] == 'o' && buf[3] == 'r' &&
		buf[4] == 'b' && buf[5] == 'i' && buf[6] == 'd' && buf[7] == 'd' &&
		buf[8] == 'e' && buf[9] == 'n' {
		blockRequest()
	}
}
