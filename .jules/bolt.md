## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Batching Cross-Day Parallelism
**Learning:** Processing daily digests day-by-day created "gaps" in concurrency. Aggregating all items across the entire time window into a single batch allows the concurrency semaphore to stay saturated, significantly reducing total wall-clock time for bulk updates.
**Action:** In multi-day processing loops, collect all work items first and then process them through a single parallel execution bottleneck.

## 2026-05-20 - Algolia Server-Side Filtering
**Learning:** Fetching 50 pages from Algolia and filtering client-side for old stories was extremely wasteful. Using `numericFilters=created_at_i>{threshold}` and early loop termination based on the `search_by_date` endpoint's chronological sort reduces API calls by ~80%.
**Action:** Always prefer server-side filtering for timestamp-based ranges and leverage the data source's sort order for early exits in polling loops.
