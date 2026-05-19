## 2026-05-10 - Resource Exhaustion via Unbounded HTTP Responses
**Vulnerability:** Potential Denial of Service (DoS) through memory exhaustion when fetching content from untrusted external URLs or APIs.
**Learning:** The application uses `io.ReadAll` on HTTP response bodies without any size restrictions.
**Prevention:** Enforce response body size limits using `io.LimitReader` for all external network requests.

## 2026-05-12 - Server-Side Request Forgery (SSRF) Risk
**Vulnerability:** External content fetching (articles, images) allowed requests to internal network addresses (loopback, private IPs).
**Learning:** Default `http.Client` does not restrict the target IP address after DNS resolution.
**Prevention:** Use a custom `net.Dialer.Control` hook to validate and block restricted IP ranges (private, loopback, link-local) before connection establishment.

## 2026-05-25 - XML and XSS Injection in Atom Feed
**Vulnerability:** The Atom feed was constructed using string concatenation with unescaped dynamic fields (siteURL, news.Summary, news.Image.URL, news.CommentURL), allowing for XML breakage and XSS in feed readers.
**Learning:** XML feeds containing HTML content blocks (like Atom with CDATA) require two levels of protection: escaping for the outer XML structure AND ensuring any dynamic data within the HTML block is also safely escaped as plain text or valid HTML.
**Prevention:** Always use `escapeXML` for all dynamic variables in `RenderFeed`, even within CDATA blocks, to ensure both XML validity and protection against XSS in consumers.
