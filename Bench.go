package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"

	"strconv"
	"sync"
	"time"
)

func registerToHandleConsoleInterrupt(startTime time.Time) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt)
	go func() {
		_ = <-signalChannel
		printResults(startTime)
		os.Exit(0)
	}()
}

func printResults(startTime time.Time) {
	var success int64
	var networkFailed int64
	var badFailed int64
	var requests int64

	for _, result := range results {
		requests += result.requests
		success += result.success
		networkFailed += result.networkFailed
		badFailed += result.badFailed
	}

	elapsed := int64(time.Since(startTime).Seconds())

	if elapsed == 0 {
		elapsed = 1
	}

	fmt.Println()
	fmt.Printf("Successful requests rate:       %10s hits/sec\n", humanize.Comma(success/elapsed))
	fmt.Println()
	fmt.Printf("Requests:                       %10s hits\n", humanize.Comma(requests))
	fmt.Printf("Successful requests:            %10s hits\n", humanize.Comma(success))
	fmt.Println()
	if networkFailed > 0 || badFailed > 0 {
		color.Set(color.FgMagenta, color.Bold)

		fmt.Printf("Network failed:                 %10s hits\n", humanize.Comma(networkFailed))
		fmt.Printf("Bad requests failed (!2xx):     %10s hits\n", humanize.Comma(badFailed))
		color.Unset()

		fmt.Println()
	}
	fmt.Printf("Read throughput:                %10s bytes/sec\n", humanize.Comma(readThroughput/elapsed))
	fmt.Printf("Write throughput:               %10s bytes/sec\n", humanize.Comma(writeThroughput/elapsed))
	fmt.Printf("Test time:                      %10s sec\n", humanize.Comma(elapsed))
}

var (
	period       int64  = 500
	writeTimeout int    = 500
	readTimeout  int    = 500
	printUrls    bool   = false
	baseUrl      string = "http://localhost:8080/databases/Bench"
)

var readThroughput int64
var writeThroughput int64
var done sync.WaitGroup

var results []*Result

func readRandomDocs(opts BenchOpts) {

	if opts.clients > opts.reads {
		opts.clients = opts.reads
	}
	done.Add(opts.clients)

	fmt.Printf("%s random doc reads from %s with %s clients\n", humanize.Comma(int64(opts.reads)), baseUrl, humanize.Comma(int64(opts.clients)))

	for i := 0; i < opts.clients; i++ {
		configuration := &Configuration{
			nextUrl: func() string {
				var id int64
				id = rand.Int63n(3594302)
				return baseUrl + "/docs?id=disks/" + strconv.FormatInt(id, 10)
			},
			method:             "GET",
			expectedStatucCode: 200,
			printUrls:          printUrls,
			requests:           int64(opts.reads / opts.clients),
		}
		results = append(results, &Result{})
		go client(configuration, results[len(results)-1], &done)
	}
}

func writeNewDocs(opts BenchOpts) {

	if opts.clients > opts.reads {
		opts.clients = opts.reads
	}
	done.Add(opts.clients)

	data, err := ioutil.ReadFile("data.json")
	if err != nil {
		color.Set(color.FgMagenta, color.Bold)

		fmt.Println("Could not read data file", err)
		color.Unset()

		os.Exit(0)
	}

	fmt.Printf("%s new doc writes from %s with %s clients\n", humanize.Comma(int64(opts.reads)), baseUrl, humanize.Comma(int64(opts.clients)))

	for i := 0; i < opts.clients; i++ {
		configuration := &Configuration{
			nextUrl: func() string {
				return baseUrl + "/docs?id=disks/"
			},
			method:             "PUT",
			expectedStatucCode: 201,
			printUrls:          printUrls,
			postData:           data,
			requests:           int64(opts.reads / opts.clients),
		}
		results = append(results, &Result{})
		go client(configuration, results[len(results)-1], &done)
	}
}

func main() {

	startTime := time.Now()

	results = []*Result{}

	registerToHandleConsoleInterrupt(startTime)

	//readRandomDocs(BenchOpts{reads: 1000 * 1000, clients: 200})
	writeNewDocs(BenchOpts{reads: 100 * 1000, clients: 200})
	fmt.Println("Waiting for results from ops...")
	done.Wait()
	printResults(startTime)
}
