package extractor

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

var (
	safeDialer = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				// If it's not an IP, it might be a hostname that hasn't been resolved yet.
				// However, Control is called after resolution for each resulting IP.
				return fmt.Errorf("invalid IP: %s", host)
			}
			if isRestrictedIP(ip) {
				return fmt.Errorf("restricted IP: %s", host)
			}
			return nil
		},
	}

	safeTransport = &http.Transport{
		DialContext:           safeDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

// GetSafeClient returns an http.Client with SSRF protection.
// It blocks requests to loopback, private, and link-local IP addresses.
// It reuses a shared transport to enable connection pooling.
func GetSafeClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: safeTransport,
		Timeout:   timeout,
	}
}

func isRestrictedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// IsPrivate reports whether ip is a private address, according to
	// RFC 1918 (IPv4 addresses) and RFC 4193 (IPv6 addresses).
	if ip.IsPrivate() {
		return true
	}

	// Defense-in-depth: explicitly block additional restricted IPv4 ranges
	ipv4 := ip.To4()
	if ipv4 != nil {
		// 0.0.0.0/8 (Local network)
		if ipv4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (Carrier-grade NAT)
		if ipv4[0] == 100 && (ipv4[1] >= 64 && ipv4[1] <= 127) {
			return true
		}
		// 198.18.0.0/15 (Benchmarking)
		if ipv4[0] == 198 && (ipv4[1] == 18 || ipv4[1] == 19) {
			return true
		}
		// 192.0.0.0/24 (IETF Protocol Assignments)
		if ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2), 203.0.113.0/24 (TEST-NET-3)
		if (ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 2) ||
			(ipv4[0] == 198 && ipv4[1] == 51 && ipv4[2] == 100) ||
			(ipv4[0] == 203 && ipv4[1] == 0 && ipv4[2] == 113) {
			return true
		}
		// 192.88.99.0/24 (6to4 Relay)
		if ipv4[0] == 192 && ipv4[1] == 88 && ipv4[2] == 99 {
			return true
		}
		// 240.0.0.0/4 (Reserved/Experimental)
		if ipv4[0] >= 240 {
			return true
		}
	}

	// IPv6 defense-in-depth
	if len(ip) == 16 {
		// 64:ff9b::/96 (NAT64)
		if ip[0] == 0 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b {
			return true
		}
		// 100::/64 (Discard-Only)
		if ip[0] == 0x01 && ip[1] == 0x00 {
			return true
		}
	}

	return false
}
