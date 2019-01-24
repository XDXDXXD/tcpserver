package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	externalAPIRateLimit = 30
	connectionCountLimit = 100
	readDeadline         = 30 * time.Second
)

type tcpResp struct {
	Data  []byte
	Error error
}

var (
	reqRate           = int64(0)
	reqCount          = int64(0)
	processedReqCount = int64(0)
	host              = flag.String("host", "localhost", "IP or localhost")
	port              = flag.Int("port", 9999, "port")
	tokenBucket       = make(chan int, externalAPIRateLimit)
	connCount         = make(chan int, connectionCountLimit)
	reqC              = make(chan int, 100000)
	processedReqC     = make(chan int, 100000)
	httpGet           = http.Get
)

func main() {
	flag.Parse()
	monitor()
	go serveTCP()
	serveHTTP()
}

func monitor() {
	// Fill up token bucket
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			for len(tokenBucket) < externalAPIRateLimit {
				tokenBucket <- 1
			}
		}
	}()

	// Reset request rate every second
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			reqRate = 0
		}
	}()

	// Calculate reqCount and reqRate
	go func() {
		for {
			<-reqC
			reqRate++
			reqCount++
		}
	}()

	// Calculate processed request count
	go func() {
		for {
			<-processedReqC
			processedReqCount++
		}
	}()
}

func serveHTTP() {
	http.HandleFunc("/tcp", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Current connection count: %d\nCurrent request rate: %d/s\nProcessed request count: %d\nRemaing jobs: %d\n", len(connCount), reqRate, reqCount, reqCount-processedReqCount)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveTCP() {
	portString := strconv.Itoa(*port)
	l, err := net.Listen("tcp", *host+":"+portString)
	if err != nil {
		log.Println("net.Listen failed", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Println("l.Close failed", err.Error())
		}
	}()
	log.Println("Listen to port:" + portString)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("l.Accept failed", err.Error())
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	connCount <- 1
	defer func() { <-connCount }()

	bufReader := bufio.NewReader(conn)
	for {
		// Set timeout = 30 seconds
		conn.SetReadDeadline(time.Now().Add(readDeadline))

		bytes, err := bufReader.ReadBytes('\n')
		if err != nil {
			log.Println("conn.Read failed", err.Error())
			continue
		}
		reqC <- 1
		sentence := string(bytes)

		if sentence == "quit\n" || sentence == "quit\r" || sentence == "quit" {
			log.Println("quit connection")
			return
		}

		select {
		case <-tokenBucket:
			// Call google to search by sentence
			data := callExternalAPI(sentence)
			// Write back search result to client
			_, err := conn.Write(data)
			if err != nil {
				log.Println("conn.Write failed", err.Error())
			}
		default:
			log.Println("Skip calling external API")
		}
		processedReqC <- 1
	}

}

func callExternalAPI(sentence string) []byte {
	fmt.Println("Calling external API")
	resp, err := httpGet("http://www.google.com/search?q=" + sentence)
	if err != nil {
		log.Println("http.Get failed", err.Error())
		return nil
	}

	if resp == nil || resp.Body == nil {
		log.Println("nil response or nil body")
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("ioutil.ReadAll failed", err.Error())
		return nil
	}
	return b
}
