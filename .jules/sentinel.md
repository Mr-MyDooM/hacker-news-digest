## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2026-05-12 - Server-Side Request Forgery (SSRF) Risk
**Vulnerability:** External content fetching (articles, images) allowed requests to internal network addresses (loopback, private IPs).
**Learning:** Default `http.Client` does not restrict the target IP address after DNS resolution.
**Prevention:** Use a custom `net.Dialer.Control` hook to validate and block restricted IP ranges (private, loopback, link-local) before connection establishment.
