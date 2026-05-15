## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2025-05-20 - Batching and Server-Side Filtering
**Learning:** Sequential processing of daily news batches created synchronization bottlenecks, and client-side filtering of API results was inefficient. Aggregating all items into a single concurrent fetch batch and utilizing server-side filtering via Algolia's `numericFilters` significantly reduces total generation time and network overhead.
**Action:** Always batch I/O-bound tasks across the entire workload to maximize concurrency utilization and prefer server-side filtering to minimize data transfer.
