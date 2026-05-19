## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Optimized Daily Fetching
**Learning:** Sequential processing of daily batches in 'GenDaily' was leaving the concurrency semaphore idle between batches. Aggregating all items into a single slice before fetching maximizes worker utilization. Additionally, Algolia's 'search_by_date' can be heavily optimized with 'numericFilters' and early termination since it is strictly chronological.
**Action:** Always batch I/O-bound items into the largest possible single set before processing. Use server-side filtering (like Algolia numericFilters) to minimize over-fetching.

## 2026-05-19 - Batch Database Operations
**Learning:** Concurrent execution of individual database queries (N+1 pattern) leads to high contention and overhead, especially with SQLite's serialized writes. Implementing batch fetches using `IN` clauses significantly reduces the number of roundtrips and lock contention. SQLite's 999 parameter limit must be handled by chunking.
**Action:** Always use batch queries (SQL `IN` clauses) with chunking (~900 items) when fetching data for a list of items to avoid the N+1 query problem.
