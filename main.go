package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	host        string
	hostsFile   string
	cidrFile    string
	concurrency int = 100
	retries     int = 5
	timeout     int = 500
	sleep       int = 100
)

func init() {
	flag.StringVar(&host, "h", "", "Single host to scan")
	flag.StringVar(&hostsFile, "hf", "", "File containing list of hosts (one per line)")
	flag.StringVar(&cidrFile, "cf", "", "File containing list of CIDR ranges (one per line)")
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

// ReadLines reads a file and returns a slice of non-empty lines
func ReadLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// ExpandCIDR takes a CIDR notation and returns all IP addresses in that range
func ExpandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// Remove network and broadcast addresses for typical use
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
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

type ScanJob struct {
	Host string
	Port int
}

type Stats struct {
	mu        sync.Mutex
	scanned   int
	openPorts int
	startTime time.Time
}

func (s *Stats) IncrementScanned() {
	s.mu.Lock()
	s.scanned++
	s.mu.Unlock()
}

func (s *Stats) IncrementOpen() {
	s.mu.Lock()
	s.openPorts++
	s.mu.Unlock()
}

func (s *Stats) GetStats() (int, int, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanned, s.openPorts, time.Since(s.startTime)
}

func worker(jobs <-chan ScanJob, wg *sync.WaitGroup, stats *Stats) {
	defer wg.Done()
	for job := range jobs {
		if TryConnect(job.Host, job.Port, retries) {
			ip, err := GetHostIP(job.Host)
			if err != nil {
				ip = job.Host
			}
			fmt.Printf("%s:%d OPEN\n", ip, job.Port)
			stats.IncrementOpen()
		}
		stats.IncrementScanned()
	}
}

func main() {
	flag.Parse()

	// Collect all hosts to scan
	var hosts []string

	// Add single host if specified
	if host != "" {
		hosts = append(hosts, host)
	}

	// Read hosts from file if specified
	if hostsFile != "" {
		fileHosts, err := ReadLines(hostsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading hosts file: %v\n", err)
			os.Exit(1)
		}
		hosts = append(hosts, fileHosts...)
	}

	// Read and expand CIDR ranges if specified
	if cidrFile != "" {
		cidrs, err := ReadLines(cidrFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading CIDR file: %v\n", err)
			os.Exit(1)
		}
		for _, cidr := range cidrs {
			ips, err := ExpandCIDR(cidr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error expanding CIDR %s: %v\n", cidr, err)
				continue
			}
			hosts = append(hosts, ips...)
		}
	}

	// Default to localhost if no hosts specified
	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1"}
	}

	totalJobs := len(hosts) * 65535
	fmt.Printf("Scanning %d host(s) across all 65535 ports (%d total combinations)...\n", len(hosts), totalJobs)

	// Create job channel for host-port combinations
	jobs := make(chan ScanJob, concurrency*10)
	var wg sync.WaitGroup

	// Initialize stats
	stats := &Stats{startTime: time.Now()}

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(jobs, &wg, stats)
	}

	// Start progress reporter
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				scanned, openPorts, elapsed := stats.GetStats()
				progress := float64(scanned) * 100 / float64(totalJobs)
				rate := float64(scanned) / elapsed.Seconds()
				eta := time.Duration(float64(totalJobs-scanned)/rate) * time.Second
				fmt.Printf("[Progress] %.2f%% | Scanned: %d/%d | Open: %d | Rate: %.0f/s | ETA: %v\n",
					progress, scanned, totalJobs, openPorts, rate, eta.Round(time.Second))
			case <-done:
				return
			}
		}
	}()

	// Generate all host-port combinations
	for _, targetHost := range hosts {
		for port := 1; port <= 65535; port++ {
			jobs <- ScanJob{Host: targetHost, Port: port}
		}
	}

	close(jobs)
	wg.Wait()
	done <- true

	scanned, openPorts, elapsed := stats.GetStats()
	fmt.Printf("\n=== Scan Complete ===\n")
	fmt.Printf("Total scanned: %d\n", scanned)
	fmt.Printf("Open ports found: %d\n", openPorts)
	fmt.Printf("Time elapsed: %v\n", elapsed.Round(time.Second))
	fmt.Printf("Average rate: %.0f ports/second\n", float64(scanned)/elapsed.Seconds())
}
