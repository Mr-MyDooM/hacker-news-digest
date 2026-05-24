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
		{"93.184.216.34", false},
		{"2606:4700:4700::1111", false},

		// Loopback
		{"127.0.0.1", true},
		{"::1", true},

		// Private IPv4 (RFC 1918)
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},

		// Private IPv6 (RFC 4193)
		{"fc00::1", true},
		{"fd00::1", true},

		// Link-local
		{"169.254.0.1", true},
		{"fe80::1", true},

		// Carrier-grade NAT (100.64.0.0/10)
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false},
		{"100.128.0.0", false},

		// Local network (0.0.0.0/8)
		{"0.0.0.0", true},
		{"0.255.255.255", true},
		{"1.0.0.0", false},

		// Benchmarking (198.18.0.0/15)
		{"198.18.0.1", true},
		{"198.19.255.255", true},
		{"198.17.255.255", false},
		{"198.20.0.0", false},

		// Unspecified
		{"0.0.0.0", true},
		{"::", true},

		// Multicast
		{"224.0.0.1", true},
		{"ff02::1", true},
	}

	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Errorf("failed to parse IP: %s", tc.ip)
			continue
		}
		if got := isRestrictedIP(ip); got != tc.expected {
			t.Errorf("isRestrictedIP(%s) = %v; want %v", tc.ip, got, tc.expected)
		}
	}
}
