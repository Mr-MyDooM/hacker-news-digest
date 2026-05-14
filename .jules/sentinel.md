## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2026-05-12 - Server-Side Request Forgery (SSRF) in External Fetches
**Vulnerability:** Attackers could provide internal URLs (e.g., localhost, private IPs) to be fetched by the server, potentially exposing internal services or cloud metadata.
**Learning:** Default `http.Client` does not restrict the destination IP, allowing connections to the internal network.
**Prevention:** Use a custom `net.Dialer` with a `Control` hook to validate and block resolved IP addresses if they fall within private or loopback ranges.
