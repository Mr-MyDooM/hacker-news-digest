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
		// Loopback
		{"127.0.0.1", true},
		{"::1", true},
		// Private
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"fd00::1", true},
		// Link-local
		{"169.254.0.1", true},
		{"fe80::1", true},
		// Multicast
		{"224.0.0.1", true},
		{"ff02::1", true},
		// Unspecified
		{"0.0.0.0", true},
		{"::", true},
		// Local network (0.0.0.0/8)
		{"0.1.2.3", true},
		// Carrier-grade NAT (100.64.0.0/10)
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		// Benchmarking (198.18.0.0/15)
		{"198.18.0.1", true},
		{"198.19.255.255", true},
		// IETF Protocol Assignments (192.0.0.0/24)
		{"192.0.0.1", true},
		// TEST-NET-1 (192.0.2.0/24)
		{"192.0.2.1", true},
		// 6to4 Relay Anycast (192.88.99.0/24)
		{"192.88.99.1", true},
		// TEST-NET-2 (198.51.100.0/24)
		{"198.51.100.1", true},
		// TEST-NET-3 (203.0.113.0/24)
		{"203.0.113.1", true},
		// Reserved (240.0.0.0/4)
		{"240.0.0.1", true},
		{"255.255.255.255", true},
		// NAT64 Well-Known Prefix (64:ff9b::/96)
		{"64:ff9b::1", true},
		// Discard-Only Address Block (100::/64)
		{"100::1", true},
		// Documentation range (2001:db8::/32)
		{"2001:db8::1", true},
		// IPv4-compatible loopback (::127.0.0.1)
		{"::127.0.0.1", true},
		// IPv4-mapped loopback/private
		{"::ffff:127.0.0.1", true},
		{"::ffff:10.0.0.1", true},
		// Public
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"20.20.20.20", false},
		{"2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
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
}
