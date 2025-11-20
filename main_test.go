package main

import (
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
		wantErr  bool
	}{
		{
			name:     "Single port",
			input:    "80",
			expected: []int{80},
			wantErr:  false,
		},
		{
			name:     "Multiple ports comma-separated",
			input:    "80,443,8080",
			expected: []int{80, 443, 8080},
			wantErr:  false,
		},
		{
			name:     "Port range",
			input:    "80-85",
			expected: []int{80, 81, 82, 83, 84, 85},
			wantErr:  false,
		},
		{
			name:     "Mixed single and range",
			input:    "22,80-82,443",
			expected: []int{22, 80, 81, 82, 443},
			wantErr:  false,
		},
		{
			name:     "Port with spaces",
			input:    "80, 443 , 8080",
			expected: []int{80, 443, 8080},
			wantErr:  false,
		},
		{
			name:     "Range with spaces",
			input:    "80 - 85",
			expected: []int{80, 81, 82, 83, 84, 85},
			wantErr:  false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Duplicate ports",
			input:    "80,80,443",
			expected: []int{80, 443},
			wantErr:  false,
		},
		{
			name:     "Overlapping ranges",
			input:    "80-85,82-87",
			expected: []int{80, 81, 82, 83, 84, 85, 86, 87},
			wantErr:  false,
		},
		{
			name:     "Invalid port - negative",
			input:    "-1",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid port - too high",
			input:    "70000",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid port - non-numeric",
			input:    "abc",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid range - start > end",
			input:    "443-80",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid range format",
			input:    "80-90-100",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Port at lower boundary",
			input:    "1",
			expected: []int{1},
			wantErr:  false,
		},
		{
			name:     "Port at upper boundary",
			input:    "65535",
			expected: []int{65535},
			wantErr:  false,
		},
		{
			name:     "Range at boundaries",
			input:    "1-5,65533-65535",
			expected: []int{1, 2, 3, 4, 5, 65533, 65534, 65535},
			wantErr:  false,
		},
		{
			name:     "Port zero - invalid",
			input:    "0",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Port 65536 - invalid",
			input:    "65536",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Complex combination",
			input:    "22,80-83,443,8000-8002,9000",
			expected: []int{22, 80, 81, 82, 83, 443, 8000, 8001, 8002, 9000},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePorts(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expected an error and got one, test passes
			if tt.wantErr {
				return
			}

			// Sort both slices for comparison (order doesn't matter in port list)
			if result != nil {
				sort.Ints(result)
			}
			if tt.expected != nil {
				sort.Ints(tt.expected)
			}

			// Compare results
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParsePorts() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExpandCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		wantErr  bool
		minCount int // minimum number of IPs expected
		maxCount int // maximum number of IPs expected
	}{
		{
			name:     "Valid /30 network",
			cidr:     "192.168.1.0/30",
			wantErr:  false,
			minCount: 2,
			maxCount: 2,
		},
		{
			name:     "Valid /29 network",
			cidr:     "192.168.1.0/29",
			wantErr:  false,
			minCount: 6,
			maxCount: 6,
		},
		{
			name:     "Valid /28 network",
			cidr:     "10.0.0.0/28",
			wantErr:  false,
			minCount: 14,
			maxCount: 14,
		},
		{
			name:     "Valid /24 network",
			cidr:     "192.168.1.0/24",
			wantErr:  false,
			minCount: 254,
			maxCount: 254,
		},
		{
			name:     "Invalid CIDR format",
			cidr:     "192.168.1.0",
			wantErr:  true,
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "Invalid IP in CIDR",
			cidr:     "999.999.999.999/24",
			wantErr:  true,
			minCount: 0,
			maxCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandCIDR(tt.cidr)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(result) < tt.minCount || len(result) > tt.maxCount {
				t.Errorf("ExpandCIDR() returned %d IPs, expected between %d and %d",
					len(result), tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestGetHostIP(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{
			name:    "Valid localhost",
			host:    "localhost",
			wantErr: false,
		},
		{
			name:    "Valid IP address",
			host:    "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "Invalid hostname",
			host:    "this-host-definitely-does-not-exist-12345.invalid",
			wantErr: true,
		},
		{
			name:    "Empty hostname",
			host:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetHostIP(tt.host)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetHostIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == "" {
				t.Errorf("GetHostIP() returned empty string for valid host")
			}
		})
	}
}

func TestReadLines(t *testing.T) {
	// Create a temporary test file
	testContent := `# This is a comment
192.168.1.1
example.com

# Another comment
10.0.0.1
`
	tmpFile := t.TempDir() + "/test_hosts.txt"
	err := os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Valid file with comments",
			filename: tmpFile,
			expected: []string{"192.168.1.1", "example.com", "10.0.0.1"},
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			filename: "/nonexistent/file.txt",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReadLines(tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ReadLines() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTryConnect(t *testing.T) {
	// Note: These tests require actual network connectivity
	// For unit tests, you might want to mock the network calls

	tests := []struct {
		name     string
		host     string
		port     int
		retries  int
		expected bool
		skip     bool
	}{
		{
			name:     "Invalid port - should fail",
			host:     "127.0.0.1",
			port:     99999,
			retries:  1,
			expected: false,
			skip:     false,
		},
		{
			name:     "Unreachable host",
			host:     "192.0.2.1", // TEST-NET-1 (RFC 5737)
			port:     80,
			retries:  1,
			expected: false,
			skip:     true, // Skip in CI/CD as it may timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping network-dependent test")
			}

			// Set short timeout for tests
			originalTimeout := timeout
			timeout = 100
			defer func() { timeout = originalTimeout }()

			result := TryConnect(tt.host, tt.port, tt.retries)
			if result != tt.expected {
				t.Errorf("TryConnect() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func BenchmarkParsePorts(b *testing.B) {
	testCases := []string{
		"80",
		"80,443,8080",
		"1-1024",
		"22,80-85,443,8000-8010",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = ParsePorts(tc)
			}
		})
	}
}
