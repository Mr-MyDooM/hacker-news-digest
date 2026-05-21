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
		{"204.79.197.200", false},
		{"2606:4700:4700::1111", false},

		// Standard restricted (Go's IsPrivate / IsLoopback etc)
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.0.1", true},
		{"fd00::1", true},

		// Newly added restricted ranges
		{"0.0.0.0", true},
		{"0.1.2.3", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"198.18.0.1", true},
		{"198.19.255.255", true},

		// Just outside restricted ranges
		{"100.63.255.255", false},
		{"100.128.0.0", false},
		{"198.17.255.255", false},
		{"198.20.0.0", false},
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
