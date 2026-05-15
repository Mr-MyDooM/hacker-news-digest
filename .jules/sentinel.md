## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2025-05-15 - Server-Side Request Forgery (SSRF) Protection
**Vulnerability:** Potential SSRF when fetching external content (articles, images) from user-provided or third-party URLs.
**Learning:** Default `http.Client` follows redirects and connects to any IP, including private network ranges, which can be exploited to probe internal services.
**Prevention:** Use a custom `net.Dialer` with a `Control` function to validate and block private, loopback, and link-local IP addresses during the connection phase.
