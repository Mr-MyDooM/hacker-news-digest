## 2026-05-10 - Parallel Content Fetching
**Learning:** Sequential network requests for content extraction and LLM summarization were a major performance bottleneck. Parallelizing these tasks with a semaphore to limit concurrency significantly reduces total generation time without overloading external servers.
**Action:** Always look for sequential I/O-bound loops that can be parallelized using Go's `sync.WaitGroup` and buffered channels as semaphores.

## 2026-05-15 - Cache-First Content Fetching
**Learning:** Network I/O was being performed even for already-cached stories because the cache check was buried inside the `Summarize` method. Moving the cache check to the beginning of `PullContent` avoids the expensive `extractor.Extract` call entirely for cached items.
**Action:** Always implement a 'cache-first' strategy before performing expensive network requests or content extraction in processing pipelines.

## 2026-05-20 - Algolia API Query Optimization
**Learning:** Client-side filtering of large API datasets is inefficient. Using Algolia's `numericFilters` to filter by date server-side drastically reduces payload size and processing time. Additionally, since the `search_by_date` endpoint is chronological, the loop can be terminated as soon as a story exceeds the age threshold.
**Action:** Prioritize server-side filtering and implement early termination in chronological data fetching loops to minimize API requests and data transfer.
