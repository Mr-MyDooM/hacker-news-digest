## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Batched Database Cache Lookup
**Learning:** Launching multiple concurrent goroutines that each perform individual database lookups for the same data type leads to an N+1 query pattern and unnecessary database connection contention. Fetching the data in a single batched query (e.g., using `WHERE IN (?)`) before starting the concurrent processing significantly reduces database overhead.
**Action:** Identify N+1 query patterns in processing pipelines and replace them with batched lookups before launching concurrent tasks.
