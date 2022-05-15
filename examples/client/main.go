package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	pb "github.com/go-kratos/examples/helloworld/helloworld"
	"google.golang.org/protobuf/proto"
)

var reqURL = "http://127.0.0.1:8080/helloworld.Greeter/SayHello"

func main() {
	hc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: false,
		},
		Timeout: time.Second * 5,
	}
	hr := &pb.HelloRequest{
		Name: "world",
	}
	b, err := proto.Marshal(hr)
	if err != nil {
		panic(err)
	}
	// grpc body:
	//   1byte 0-uncompressed, 1-compressed using the mechanism declared by the Message-Encoding header.
	//   4byte message length
	//   bytes message data
	var head [5]byte
	binary.BigEndian.PutUint32(head[1:], uint32(len(b)))
	buff := bytes.NewBuffer(nil)
	buff.Write(head[:])
	buff.Write(b)
	fmt.Println(buff.Bytes())
	req, err := http.NewRequest(http.MethodPost, reqURL, buff)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/grpc")
	resp, err := hc.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println("StatusCode:", resp.StatusCode)
	for k, v := range resp.Header {
		fmt.Printf("Header: %s: %s\n", k, v)
	}
	for k, v := range resp.Trailer {
		fmt.Printf("Trailer: %s: %s\n", k, v)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("resp body:", b)
}
