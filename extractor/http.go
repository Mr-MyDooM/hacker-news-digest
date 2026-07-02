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
	cidrs := []string{
		"192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.51.100.0/24",
		"203.0.113.0/24", "240.0.0.0/4", "100.64.0.0/10", "198.18.0.0/15",
		"2001:10::/28", "2001:20::/28", "64:ff9b::/96", "100::/64",
	}
	for _, cidr := range cidrs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		restrictedNetworks = append(restrictedNetworks, n)
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
				return fmt.Errorf("invalid protocol: %s", req.URL.Scheme)
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
	// IsPrivate reports whether ip is a private address, according to
	// RFC 1918 (IPv4 addresses) and RFC 4193 (IPv6 addresses).
	if ip.IsPrivate() {
		return true
	}

	// Defense-in-depth: explicitly block additional restricted IP ranges
	ipv4 := ip.To4()
	if ipv4 != nil && ipv4[0] == 0 { // 0.0.0.0/8 (Local network)
		return true
	}

	for _, n := range restrictedNetworks {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}
