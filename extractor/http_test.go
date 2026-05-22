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
		{"2606:4700:4700::1111", false},

		// Loopback
		{"127.0.0.1", true},
		{"::1", true},

		// Private IPs
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"fd00::1", true},

		// Link-local
		{"169.254.0.1", true},
		{"fe80::1", true},

		// CGNAT (100.64.0.0/10) - Currently missing
		{"100.64.0.1", true},
		{"100.127.255.255", true},

		// Benchmarking (198.18.0.0/15) - Currently missing
		{"198.18.0.1", true},
		{"198.19.255.255", true},

		// Local network (0.0.0.0/8) - Currently missing
		{"0.0.0.1", true},
		{"0.255.255.255", true},
		{"0.0.0.0", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Errorf("failed to parse IP: %s", tt.ip)
			continue
		}
		got := isRestrictedIP(ip)
		if got != tt.expected {
			t.Errorf("isRestrictedIP(%s) = %v; want %v", tt.ip, got, tt.expected)
		}
	}
}
