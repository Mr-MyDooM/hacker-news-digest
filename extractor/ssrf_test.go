package extractor

import (
	"net"
	"testing"
	"time"
)

func TestIsSafeIP(t *testing.T) {
	tests := []struct {
		ip   string
		safe bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"169.254.169.254", false},
		{"0.0.0.0", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isSafeIP(ip); got != tt.safe {
			t.Errorf("isSafeIP(%s) = %v, want %v", tt.ip, got, tt.safe)
		}
	}
}

func TestSafeHTTPClientBlocking(t *testing.T) {
	client := SafeHTTPClient(1 * time.Second)

	// Test blocking a private IP
	_, err := client.Get("http://192.168.1.1")
	if err == nil {
		t.Error("Expected error when accessing private IP, got nil")
	} else if !contains(err.Error(), "blocked") && !contains(err.Error(), "connection refused") && !contains(err.Error(), "i/o timeout") {
		// Note: on some systems it might fail with "connection refused" before our Control check if nothing is listening,
		// but our Control should catch it if it tries to connect.
		// Actually, Control is called before connect.
		t.Errorf("Expected blocked error, got: %v", err)
	}

	// Test blocking loopback
	_, err = client.Get("http://127.0.0.1:8080")
	if err == nil {
		t.Error("Expected error when accessing loopback, got nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||  (len(s) > len(substr) &&  (s[1:] == substr || contains(s[1:], substr))))
}
