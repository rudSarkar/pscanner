package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"time"
)

var host string

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan")
}

// TryConnect attempts to connect to a single port with retries
func TryConnect(host string, port int, retries int) bool {
	address := fmt.Sprintf("%s:%d", host, port)

	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond) // avoid hammering the host
	}
	return false
}

func worker(host string, ports <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for port := range ports {
		if TryConnect(host, port, 5) {
			fmt.Printf("Port %d OPEN\n", port)
		}
	}
}

func main() {
	flag.Parse()

	portChan := make(chan int, 100)
	var wg sync.WaitGroup

	// Limit concurrency to avoid flooding (adjust as needed)
	workerCount := 200

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(host, portChan, &wg)
	}

	// Send ports 1-65535 into the worker queue
	for port := 1; port <= 65535; port++ {
		portChan <- port
	}

	close(portChan)
	wg.Wait()

	fmt.Println("Scan complete.")
}
