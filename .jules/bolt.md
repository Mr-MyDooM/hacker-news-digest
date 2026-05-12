## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Early Exit for Paginated API Calls
**Learning:** The Algolia `search_by_date` endpoint returns results in strict descending order of creation time. In `GetDailyNews`, we were fetching up to 50 pages of results regardless of the target date range. Implementing an early break when encountering a story older than the threshold avoids dozens of redundant network requests.
**Action:** When consuming paginated APIs with a known sort order, always implement an early exit (break/return) once the threshold condition is met to minimize unnecessary I/O.
