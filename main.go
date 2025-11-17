package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	host        string
	concurrency int = 100
	retries     int = 5
	timeout     int = 500
	sleep       int = 100
)

func init() {
	flag.StringVar(&host, "h", "127.0.0.1", "Host to scan")
	flag.IntVar(&concurrency, "c", 100, "Number of concurrent workers")
	flag.IntVar(&retries, "r", 5, "Number of retries for each port")
	flag.IntVar(&timeout, "t", 500, "Connection timeout in milliseconds")
	flag.IntVar(&sleep, "s", 100, "Sleep time between retries in milliseconds")
}

func GetHostIP(host string) (string, error) {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("unable to resolve host: %s", host)
	}
	return ips[0].String(), nil
}

// TryConnect attempts to connect to a single port with retries
func TryConnect(host string, port int, retries int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Duration(timeout)*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(time.Duration(sleep) * time.Millisecond) // avoid hammering the host
	}
	return false
}

func worker(host string, ports <-chan int, wg *sync.WaitGroup) {
	ip, err := GetHostIP(host)
	if err != nil {
		ip = host
	}

	defer wg.Done()
	for port := range ports {
		if TryConnect(host, port, retries) {
			fmt.Printf("%s:%d OPEN\n", ip, port)
		}
	}
}

func main() {
	flag.Parse()

	portChan := make(chan int, 100)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
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
