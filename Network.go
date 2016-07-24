package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
	"sync"
	"sync/atomic"
)

type BenchOpts struct {
	reads   int
	clients int
}

type Configuration struct {
	nextUrl            func() string
	method             string
	postData           []byte
	requests           int64
	period             int64
	expectedStatucCode int
	myClient           fasthttp.Client
	printUrls          bool
}

type Result struct {
	requests      int64
	success       int64
	networkFailed int64
	badFailed     int64
}

type MyConn struct {
	net.Conn
}

func (this *MyConn) Read(b []byte) (n int, err error) {
	len, err := this.Conn.Read(b)

	if err == nil {
		atomic.AddInt64(&readThroughput, int64(len))
	}

	return len, err
}

func (this *MyConn) Write(b []byte) (n int, err error) {
	len, err := this.Conn.Write(b)

	if err == nil {
		atomic.AddInt64(&writeThroughput, int64(len))
	}

	return len, err
}

func MyDialer() func(address string) (conn net.Conn, err error) {
	return func(address string) (net.Conn, error) {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return nil, err
		}

		myConn := &MyConn{Conn: conn}

		return myConn, nil
	}
}
func client(configuration *Configuration, result *Result, done *sync.WaitGroup) {
	configuration.myClient.Dial = MyDialer()
	for result.requests < configuration.requests {

		var tmpUrl = configuration.nextUrl()
		req := fasthttp.AcquireRequest()
		req.SetRequestURI(tmpUrl)

		if configuration.printUrls {
			fmt.Println(tmpUrl)
		}

		req.Header.SetMethodBytes([]byte(configuration.method))

		req.Header.Set("Connection", "keep-alive")

		if configuration.postData != nil {
			req.SetBody(configuration.postData)
		}

		resp := fasthttp.AcquireResponse()
		err := configuration.myClient.Do(req, resp)
		statusCode := resp.StatusCode()
		result.requests++
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if err != nil {
			fmt.Println(err)
			result.networkFailed++
			continue
		}

		if statusCode == configuration.expectedStatucCode {
			result.success++
		} else {
			result.badFailed++
		}
	}

	done.Done()
}
