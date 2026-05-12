## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2026-05-13 - Server-Side Request Forgery (SSRF) in Content Extraction
**Vulnerability:** The application could be used to probe internal network services by providing a malicious URL that resolves to a private IP address.
**Learning:** Standard `http.Client` does not restrict the destination IP, allowing requests to localhost or internal infrastructure.
**Prevention:** Use a custom `net.Dialer` with a `Control` function to validate and block connections to non-public IP addresses before the connection is established.
