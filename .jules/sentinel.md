## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2026-05-15 - SSRF via External Content Fetching
**Vulnerability:** Server-Side Request Forgery (SSRF) when fetching external article content or images, allowing attackers to probe internal networks.
**Learning:** Standard `http.Client` does not restrict destination IP addresses, which can include sensitive local or private network ranges.
**Prevention:** Use a custom `net.Dialer.Control` hook to validate and block connections to private, loopback, and link-local IP addresses after DNS resolution.
