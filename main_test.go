package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	clientNum := 10
	queryNum := 3
	done := make(chan int, clientNum)
	// Use mock external api
	httpGet = func(url string) (*http.Response, error) {
		s := strings.SplitAfter(url, "=")
		return &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(s[1])))}, nil
	}
	// Run server
	go main()

	time.Sleep(5 * time.Second)

	// Run 100 clients
	for i := 0; i < clientNum; i++ {
		go func(i int) {
			conn, err := net.Dial("tcp", "localhost:9999")
			if err != nil {
				t.Error("net.Dail failed", err.Error())
				return
			}
			conn.SetReadDeadline(time.Now().Add(readDeadline))

			// Send quernNum queries
			for q := 1; q <= queryNum; q++ {
				text := strconv.Itoa(i) + ":" + strconv.Itoa(q) + "\n"

				_, err = conn.Write([]byte(text))
				if err != nil {
					t.Error("bufWriter.WriteString failed", err.Error(), i, q)
					continue
				}

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					t.Error("bufReader.ReadBytes failed", err.Error(), i, q)
					continue
				}

				if string(buf[:n]) != text {
					t.Error("Unexpected response", text, string(buf[:n]))
					continue
				}
			}
			done <- 1
		}(i)
	}

	for i := 0; i < clientNum; i++ {
		<-done
	}
}
