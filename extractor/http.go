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

	restrictedNetworks []*net.IPNet
)

func init() {
	cidrs := []string{
		"0.0.0.0/8",          // Local network
		"100.64.0.0/10",      // Shared Address Space (CGNAT)
		"192.0.0.0/24",       // IETF Protocol Assignments
		"192.0.2.0/24",       // Documentation (TEST-NET-1)
		"198.18.0.0/15",      // Benchmarking
		"198.51.100.0/24",    // Documentation (TEST-NET-2)
		"203.0.113.0/24",     // Documentation (TEST-NET-3)
		"240.0.0.0/4",        // Reserved
		"255.255.255.255/32", // Limited Broadcast
		"2001:10::/28",       // ORCHID
		"2001:20::/28",       // ORCHIDv2
		"2001:db8::/32",      // Documentation (IPv6)
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("failed to parse CIDR %s: %v", cidr, err))
		}
		restrictedNetworks = append(restrictedNetworks, network)
	}
}

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
	if ip == nil {
		return true // Fail secure
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// IsPrivate reports whether ip is a private address, according to
	// RFC 1918 (IPv4 addresses) and RFC 4193 (IPv6 addresses).
	if ip.IsPrivate() {
		return true
	}

	for _, network := range restrictedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
