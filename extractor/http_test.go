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
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"169.254.0.1", true}, // Link-local
		{"8.8.8.8", false},    // Public
		{"1.1.1.1", false},    // Public
		{"100.64.0.1", true},  // CGNAT
		{"198.18.0.1", true},  // Benchmarking
		{"0.0.0.1", true},     // 0.0.0.0/8
		{"::1", true},         // IPv6 loopback
		{"fe80::1", true},     // IPv6 link-local
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP: %s", tt.ip)
		}
		if got := isRestrictedIP(ip); got != tt.expected {
			t.Errorf("isRestrictedIP(%s) = %v; want %v", tt.ip, got, tt.expected)
		}
	}
}
