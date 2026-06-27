package extractor

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

var (
	restrictedNetworks []*net.IPNet

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

func init() {
	// Comprehensive list of restricted CIDR ranges for SSRF defense-in-depth.
	cidrs := []string{
		"0.0.0.0/8",          // Local network
		"10.0.0.0/8",         // Private-Use (RFC 1918)
		"100.64.0.0/10",      // Carrier-grade NAT
		"127.0.0.0/8",        // Loopback
		"169.254.0.0/16",     // Link-local
		"172.16.0.0/12",      // Private-Use (RFC 1918)
		"192.0.0.0/24",       // IETF Protocol Assignments
		"192.0.2.0/24",       // TEST-NET-1
		"192.88.99.0/24",      // 6to4 Relay
		"192.168.0.0/16",     // Private-Use (RFC 1918)
		"198.18.0.0/15",      // Benchmarking
		"198.51.100.0/24",    // TEST-NET-2
		"203.0.113.0/24",     // TEST-NET-3
		"224.0.0.0/4",        // Multicast
		"240.0.0.0/4",        // Reserved
		"255.255.255.255/32", // Limited Broadcast

		// IPv6
		"::/128",          // Unspecified
		"::1/128",         // Loopback
		"64:ff9b::/96",    // NAT64
		"100::/64",        // Discard-Only Address Block
		"2001:10::/28",    // ORCHIDv2
		"2001:20::/28",    // ORCHIDv2
		"2001:db8::/32",   // Documentation
		"fc00::/7",        // Unique-Local
		"fe80::/10",       // Link-Local Unicast
		"ff00::/8",        // Multicast
	}

	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("failed to parse restricted CIDR %s: %v", cidr, err))
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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Only allow http/https
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("invalid redirect scheme: %s", req.URL.Scheme)
			}
			return nil
		},
	}
}

func isRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true // Fail secure
	}

	// Standard library checks
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}

	// CIDR-based checks for defense-in-depth and additional ranges
	for _, network := range restrictedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
