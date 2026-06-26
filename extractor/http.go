package extractor

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

var (
	restrictedCIDRs []*net.IPNet

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

func init() {
	cidrs := []string{
		// IPv4
		"0.0.0.0/8",          // Local network
		"100.64.0.0/10",      // Carrier-grade NAT
		"192.0.0.0/24",       // IETF Protocol Assignments
		"192.0.2.0/24",       // TEST-NET-1
		"192.88.99.0/24",     // 6to4 Relay
		"198.18.0.0/15",      // Benchmarking
		"198.51.100.0/24",    // TEST-NET-2
		"203.0.113.0/24",     // TEST-NET-3
		"240.0.0.0/4",        // Reserved
		"255.255.255.255/32", // Limited Broadcast

		// IPv6
		"100::/64",       // Discard-Only Prefix
		"2001:db8::/32",  // Documentation
		"2001:10::/28",   // ORCHIDv2
		"2001:20::/28",   // ORCHIDv2
		"64:ff9b::/96",   // Well-Known Prefix for NAT64
	}
	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("failed to parse CIDR %s: %v", cidr, err))
		}
		restrictedCIDRs = append(restrictedCIDRs, block)
	}
}

// GetSafeClient returns an http.Client with SSRF protection.
// It blocks requests to loopback, private, and link-local IP addresses.
// It reuses a shared transport to enable connection pooling.
func GetSafeClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: safeTransport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("invalid redirect protocol: %s", req.URL.Scheme)
			}
			return nil
		},
	}
}

func isRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true // Fail secure
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}

	for _, block := range restrictedCIDRs {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}
