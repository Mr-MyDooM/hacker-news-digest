## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Optimized Daily Fetching
**Learning:** Sequential processing of daily batches in 'GenDaily' was leaving the concurrency semaphore idle between batches. Aggregating all items into a single slice before fetching maximizes worker utilization. Additionally, Algolia's 'search_by_date' can be heavily optimized with 'numericFilters' and early termination since it is strictly chronological.
**Action:** Always batch I/O-bound items into the largest possible single set before processing. Use server-side filtering (like Algolia numericFilters) to minimize over-fetching.

## 2026-05-25 - Batch Database Lookups (N+1 Query Resolution)
**Learning:** Performing individual database queries for each item in a concurrent processing loop (N+1 problem) causes significant overhead due to network/IPC latency and database lock contention, even with SQLite. Prefetching data using SQL 'IN' clauses with chunking (to respect parameter limits) is much more efficient.
**Action:** Always prefetch cacheable data in bulk before entering concurrent loops. Use anonymous functions or explicit 'rows.Close()' when executing chunked queries in a loop to avoid resource leaks.
