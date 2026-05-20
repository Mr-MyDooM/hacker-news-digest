package extractor

import (
	"net"
	"testing"
)

func TestIsRestrictedIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		// Public IPs
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"209.85.145.139", false},

		// Loopback
		{"127.0.0.1", true},
		{"::1", true},

		// Private ranges (RFC 1918)
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},

		// Link-local
		{"169.254.169.254", true},

		// Unspecified
		{"0.0.0.0", true},

		// CGNAT (RFC 6598)
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false},

		// Benchmarking (RFC 2544)
		{"198.18.0.1", true},
		{"198.19.255.255", true},
		{"198.20.0.1", false},

		// 0.0.0.0/8
		{"0.1.2.3", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP %s", tt.ip)
		}
		got := isRestrictedIP(ip)
		if got != tt.expected {
			t.Errorf("isRestrictedIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}
