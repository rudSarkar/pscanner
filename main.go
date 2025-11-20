package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	host        string
	hostsFile   string
	cidrFile    string
	ports       string
	outputFile  string
	concurrency int = 100
	retries     int = 5
	timeout     int = 500
	sleep       int = 100
)

func init() {
	flag.StringVar(&host, "h", "", "Single host to scan")
	flag.StringVar(&hostsFile, "hf", "", "File containing list of hosts (one per line)")
	flag.StringVar(&cidrFile, "cf", "", "File containing list of CIDR ranges (one per line)")
	flag.StringVar(&ports, "p", "", "Ports to scan (e.g., 80, 80-443, 80,443,8080)")
	flag.StringVar(&outputFile, "o", "", "Output file to save results")
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

// ParsePorts parses port specification and returns a list of ports
// Supports:
// - Single port: "80"
// - Range: "80-443"
// - Comma-separated: "80,443,8080"
// - Combination: "80,443-445,8080"
func ParsePorts(portSpec string) ([]int, error) {
	if portSpec == "" {
		return nil, nil
	}

	var ports []int
	portSet := make(map[int]bool)

	// Split by comma
	parts := strings.Split(portSpec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", part)
			}
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", rangeParts[1])
			}
			if start < 1 || start > 65535 || end < 1 || end > 65535 {
				return nil, fmt.Errorf("port numbers must be between 1 and 65535")
			}
			if start > end {
				return nil, fmt.Errorf("invalid range: start port > end port")
			}
			for p := start; p <= end; p++ {
				portSet[p] = true
			}
		} else {
			// Single port
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", part)
			}
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("port number must be between 1 and 65535")
			}
			portSet[port] = true
		}
	}

	// Convert map to sorted slice
	for port := range portSet {
		ports = append(ports, port)
	}

	return ports, nil
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
	output    io.Writer
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
			result := fmt.Sprintf("%s:%d\n", ip, job.Port)
			fmt.Print(result)
			if stats.output != nil {
				stats.output.Write([]byte(result))
			}
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

	// Parse ports
	var portList []int
	if ports != "" {
		var err error
		portList, err = ParsePorts(ports)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing ports: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Default to all ports
		for p := 1; p <= 65535; p++ {
			portList = append(portList, p)
		}
	}

	totalJobs := len(hosts) * len(portList)
	fmt.Printf("Scanning %d host(s) across %d ports (%d total combinations)...\n", len(hosts), len(portList), totalJobs)

	// Create job channel for host-port combinations
	jobs := make(chan ScanJob, concurrency*10)
	var wg sync.WaitGroup

	// Initialize stats and output writer
	var outputWriter io.Writer
	var outputFileHandle *os.File
	if outputFile != "" {
		var err error
		outputFileHandle, err = os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer outputFileHandle.Close()
		outputWriter = outputFileHandle
		fmt.Printf("Output will be saved to: %s\n", outputFile)
	}

	stats := &Stats{startTime: time.Now(), output: outputWriter}

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
		for _, port := range portList {
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
