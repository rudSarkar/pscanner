# Port Scanner

A fast, concurrent TCP port scanner written in Go that supports scanning single hosts, multiple hosts from a file, or entire CIDR ranges.

## Features

- 🚀 Fast concurrent scanning with configurable workers
- 🎯 Flexible port specification (single, range, or comma-separated)
- 💾 Save results to output file
- 🔄 Automatic retry mechanism for reliable results
- 📊 Real-time progress reporting with ETA
- 🌐 Support for hostnames and IP addresses
- 📝 Batch scanning from host files
- 🔢 CIDR range expansion support
- ⚙️ Configurable timeouts and retry delays

## Installation

```bash
go install github.com/rudSarkar/pscanner@latest
```

## Usage

### Basic Usage

```bash
# Scan a single host (all 65535 ports)
pscanner -h example.com

# Scan specific ports
pscanner -h example.com -p 80,443,8080

# Scan port range
pscanner -h example.com -p 1-1024

# Scan with custom concurrency
pscanner -h example.com -c 200

# Save output to file
pscanner -h example.com -p 80-443 -o results.txt
```

### Scanning Multiple Hosts

Create a file with one host per line:

```bash
# hosts.txt
192.168.1.1
example.com
google.com
```

Then run:

```bash
pscanner -hf hosts.txt
```

### Scanning CIDR Ranges

Create a file with CIDR ranges:

```bash
# cidrs.txt
192.168.1.0/24
10.0.0.0/28
```

Then run:

```bash
pscanner -cf cidrs.txt
```

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-h` | Single host to scan | "" |
| `-hf` | File containing list of hosts (one per line) | "" |
| `-cf` | File containing list of CIDR ranges (one per line) | "" |
| `-p` | Ports to scan (e.g., 80, 80-443, 80,443,8080) | All ports (1-65535) |
| `-o` | Output file to save results | "" |
| `-c` | Number of concurrent workers | 100 |
| `-r` | Number of retries for each port | 5 |
| `-t` | Connection timeout in milliseconds | 500 |
| `-s` | Sleep time between retries in milliseconds | 100 |

### Examples

```bash
# Scan common web ports
pscanner -h example.com -p 80,443,8080,8443

# Scan well-known ports (1-1024)
pscanner -h example.com -p 1-1024

# Fast scan with high concurrency
pscanner -h 192.168.1.1 -c 500

# Slower, more reliable scan
pscanner -h example.com -c 50 -r 10 -t 1000

# Scan multiple targets with custom settings and save output
pscanner -hf hosts.txt -p 22,80,443 -c 200 -o scan_results.txt

# Scan CIDR range on specific ports
pscanner -cf cidrs.txt -p 22,3389 -c 150

# Combine port ranges and individual ports
pscanner -h example.com -p 20-25,80,443-445,3389
```

## Output

The scanner provides:

- **Real-time progress updates** every 5 seconds showing:
  - Progress percentage
  - Ports scanned vs total
  - Number of open ports found
  - Scanning rate (ports/second)
  - Estimated time to completion

- **Open port notifications** as they are discovered:
  ```
  192.168.1.1:22
  192.168.1.1:80
  192.168.1.1:443
  ```

- **Final summary** with total statistics

### Sample Output

```
Scanning 1 host(s) across 100 ports (100 total combinations)...
Output will be saved to: results.txt
192.168.1.1:22
192.168.1.1:80
[Progress] 45.00% | Scanned: 45/100 | Open: 2 | Rate: 15/s | ETA: 3s
192.168.1.1:443
[Progress] 90.00% | Scanned: 90/100 | Open: 3 | Rate: 18/s | ETA: 0s
...

=== Scan Complete ===
Total scanned: 100
Open ports found: 3
Time elapsed: 6s
Average rate: 16 ports/second
```

## Performance Tips

- **Increase concurrency** (`-c`) for faster scans, but be aware of system limits and network constraints
- **Reduce retries** (`-r`) if you're confident in network stability
- **Lower timeout** (`-t`) for faster scanning of responsive hosts
- **Reduce sleep time** (`-s`) between retries if network is reliable

## Notes

- By default, the scanner attempts all 65535 TCP ports for each host unless `-p` flag is specified
- Use the `-p` flag to target specific ports for faster, focused scans
- Network and broadcast addresses are excluded when expanding CIDR ranges
- Results are displayed in real-time and optionally saved to a file with `-o`
- The tool requires appropriate network permissions to scan hosts

## License

This project is provided as-is for educational and legitimate network administration purposes. Always ensure you have permission before scanning networks you don't own.
