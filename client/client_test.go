package client

import (
	"fmt"
	"testing"
)

func TestRequestHTTPS(t *testing.T) {
	resp, err := _globalHTTPSClient.Get("https://www.bilibili.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println("RESP", resp.Proto, resp.Status, resp.StatusCode)
}
