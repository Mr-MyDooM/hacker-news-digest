package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestSafeHTTPClient(t *testing.T) {
	client := GetSafeClient(2 * time.Second)

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://127.0.0.1", true},
		{"http://localhost", true},
		{"http://192.168.1.1", true},
		{"http://10.0.0.1", true},
		{"http://172.16.0.1", true},
		{"http://169.254.169.254", true}, // Link-local
		{"https://www.google.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, err := client.Get(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Get(%s) expected error, got nil", tt.url)
				} else {
					t.Logf("Get(%s) got expected error: %v", tt.url, err)
				}
			} else {
				// We don't care if it actually succeeds (might be no internet),
				// just that it doesn't fail due to the SSRF check.
				// However, if it fails with "connection to ... is blocked", that's a failure.
				if err != nil && (strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "private")) {
					t.Errorf("Get(%s) unexpected SSRF block: %v", tt.url, err)
				}
			}
		})
	}
}
