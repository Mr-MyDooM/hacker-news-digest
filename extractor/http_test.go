package extractor

import (
	"net"
	"testing"
)

func TestIsRestrictedIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Loopback
		{"Loopback IPv4", "127.0.0.1", true},
		{"Loopback IPv6", "::1", true},
		// Private
		{"Private 10.x", "10.0.0.1", true},
		{"Private 172.16.x", "172.16.0.1", true},
		{"Private 192.168.x", "192.168.1.1", true},
		{"Private IPv6 (ULA)", "fd00::1", true},
		// Link-local
		{"Link-local IPv4", "169.254.0.1", true},
		{"Link-local IPv6", "fe80::1", true},
		// Multicast
		{"Multicast IPv4", "224.0.0.1", true},
		{"Multicast IPv6", "ff02::1", true},
		// Unspecified
		{"Unspecified IPv4", "0.0.0.0", true},
		{"Unspecified IPv6", "::", true},
		// Local network (0.0.0.0/8)
		{"Local network", "0.1.2.3", true},
		// Carrier-grade NAT (100.64.0.0/10)
		{"CGNAT", "100.64.0.1", true},
		{"CGNAT end", "100.127.255.255", true},
		// Benchmarking (198.18.0.0/15)
		{"Benchmarking start", "198.18.0.1", true},
		{"Benchmarking end", "198.19.255.255", true},
		// Documentation
		{"TEST-NET-1", "192.0.2.1", true},
		{"TEST-NET-2", "198.51.100.1", true},
		{"TEST-NET-3", "203.0.113.1", true},
		{"IPv6 documentation", "2001:db8::1", true},
		// ORCHID
		{"ORCHID", "2001:10::1", true},
		{"ORCHIDv2", "2001:20::1", true},
		// Public
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Microsoft IP", "20.20.20.20", false},
		{"Google DNS IPv6", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			got := isRestrictedIP(ip)
			if got != tt.expected {
				t.Errorf("isRestrictedIP(%s) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}

	t.Run("nil IP", func(t *testing.T) {
		if !isRestrictedIP(nil) {
			t.Error("isRestrictedIP(nil) = false, want true")
		}
	})
}
