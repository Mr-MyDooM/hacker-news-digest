package extractor

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// SafeHTTPClient is an http.Client configured with a custom dialer that
// prevents Server-Side Request Forgery (SSRF) by blocking connections to
// private, loopback, and link-local IP addresses.
var SafeHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
			Control:   isSafeAddress,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func isSafeAddress(network, address string, c syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", host)
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return fmt.Errorf("connection to loopback or link-local address is blocked: %s", host)
	}

	if ip.IsPrivate() {
		return fmt.Errorf("connection to private network address is blocked: %s", host)
	}

	return nil
}

// GetSafeClient returns a copy of SafeHTTPClient with a custom timeout.
func GetSafeClient(timeout time.Duration) *http.Client {
	client := *SafeHTTPClient
	client.Timeout = timeout
	return &client
}
