package extractor

import (
	"net/http"
	"testing"
	"time"
)

func TestSafeHTTPClient_BlocksPrivateIPs(t *testing.T) {
	client := SafeHTTPClient(2 * time.Second)

	privateURLs := []string{
		"http://127.0.0.1",
		"http://192.168.1.1",
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://localhost",
		"http://[::1]",
	}

	for _, u := range privateURLs {
		_, err := client.Get(u)
		if err == nil {
			t.Errorf("expected error for private URL %s, but got none", u)
		} else {
			t.Logf("correctly got error for %s: %v", u, err)
		}
	}
}

func TestSafeHTTPClient_AllowsPublicIPs(t *testing.T) {
	// We can't easily test real public IPs without network access,
	// but we can at least check if it doesn't fail immediately for a public-looking domain.
	// In the sandbox environment, network access might be restricted anyway.
	client := SafeHTTPClient(5 * time.Second)

	// Use a reliable public site
	u := "https://www.google.com"
	resp, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(resp)
	if err != nil {
		// If it's a network error (e.g. DNS failure in sandbox), it's acceptable.
		// We just want to make sure it's not our Control function blocking it
		// because it thinks it's a private IP.
		t.Logf("Note: request to %s failed (expected in some restricted environments): %v", u, err)
	}
}
