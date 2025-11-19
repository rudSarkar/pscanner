# Port Scanner

A fast, concurrent TCP port scanner written in Go that supports scanning single hosts, multiple hosts from a file, or entire CIDR ranges.

## Features

- 🚀 Fast concurrent scanning with configurable workers
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

# Scan localhost
pscanner -h 127.0.0.1

# Scan with custom concurrency
pscanner -h example.com -c 200
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
| `-c` | Number of concurrent workers | 100 |
| `-r` | Number of retries for each port | 5 |
| `-t` | Connection timeout in milliseconds | 500 |
| `-s` | Sleep time between retries in milliseconds | 100 |

### Examples

```bash
# Fast scan with high concurrency
pscanner -h 192.168.1.1 -c 500

# Slower, more reliable scan
pscanner -h example.com -c 50 -r 10 -t 1000

# Scan multiple targets with custom settings
pscanner -hf hosts.txt -c 200 -t 300 -s 50

# Scan CIDR range
pscanner -cf cidrs.txt -c 150
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
  192.168.1.1:22 OPEN
  192.168.1.1:80 OPEN
  192.168.1.1:443 OPEN
  ```

- **Final summary** with total statistics

### Sample Output

```
Scanning 1 host(s) across all 65535 ports (65535 total combinations)...
192.168.1.1:22 OPEN
192.168.1.1:80 OPEN
[Progress] 15.23% | Scanned: 9984/65535 | Open: 2 | Rate: 1997/s | ETA: 27s
192.168.1.1:443 OPEN
[Progress] 30.46% | Scanned: 19968/65535 | Open: 3 | Rate: 1998/s | ETA: 22s
...

=== Scan Complete ===
Total scanned: 65535
Open ports found: 3
Time elapsed: 33s
Average rate: 1986 ports/second
```

## Performance Tips

- **Increase concurrency** (`-c`) for faster scans, but be aware of system limits and network constraints
- **Reduce retries** (`-r`) if you're confident in network stability
- **Lower timeout** (`-t`) for faster scanning of responsive hosts
- **Reduce sleep time** (`-s`) between retries if network is reliable

## Notes

- The scanner attempts all 65535 TCP ports for each host
- Network and broadcast addresses are excluded when expanding CIDR ranges
- The tool requires appropriate network permissions to scan hosts

## License

This project is provided as-is for educational and legitimate network administration purposes. Always ensure you have permission before scanning networks you don't own.
