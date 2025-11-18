package motor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// SearchOptions configures search behavior
type SearchOptions struct {
	Mode               SearchMode // plaintext or regex
	SearchResponseBody bool       // deep search flag (default: false)
	WorkerCount        int        // default: runtime.numcpu()
	ChunkSize          int        // entries per work batch (default: 0 = auto-partition)
}

// DefaultSearchOptions provides sensible defaults
var DefaultSearchOptions = SearchOptions{
	Mode:               PlainText,
	SearchResponseBody: false,
	WorkerCount:        runtime.NumCPU(),
	ChunkSize:          0, // auto-partition
}

// SearchResult represents a single match
type SearchResult struct {
	Index int    // entry index in har file
	Field string // which field matched: "url", "request.body", "response.headers.content-type"
	Error error  // non-fatal error reading this entry (search continues)
}

// SearchStats tracks search performance metrics
type SearchStats struct {
	EntriesSearched int64         // total entries processed
	MatchesFound    int64         // total matches found
	BytesSearched   int64         // total bytes read from disk
	SearchDuration  time.Duration // total search time
}

// searchAtomicStats holds search statistics with atomic operations
type searchAtomicStats struct {
	entriesSearched int64
	matchesFound    int64
	bytesSearched   int64
	searchDuration  int64 // nanoseconds
}

// HARSearcher provides efficient search across har entries
type HARSearcher struct {
	streamer   HARStreamer
	reader     EntryReader
	bufferPool *sync.Pool
	stats      searchAtomicStats
}

// creates a new har searcher
func NewSearcher(streamer HARStreamer, reader EntryReader) *HARSearcher {
	return &HARSearcher{
		streamer: streamer,
		reader:   reader,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 64*1024) // 64kb buffers
				return &buf
			},
		},
	}
}

// Search executes a search and streams results via channel
func (s *HARSearcher) Search(ctx context.Context, pattern string, opts SearchOptions) (<-chan []SearchResult, error) {
	// set defaults
	if opts.WorkerCount == 0 {
		opts.WorkerCount = runtime.NumCPU()
	}

	// compile pattern once (not per entry!)
	compiledPattern, err := compilePattern(pattern, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// get total entries
	index := s.streamer.GetIndex()
	totalEntries := index.TotalEntries

	// handle empty har
	if totalEntries == 0 {
		emptyResults := make(chan []SearchResult)
		close(emptyResults)
		return emptyResults, nil
	}

	// create work batches
	batches := createWorkBatches(totalEntries, opts)

	// create channels
	workQueue := make(chan workBatch, opts.WorkerCount*2)
	results := make(chan []SearchResult, opts.WorkerCount)

	// start timer
	startTime := time.Now()

	// spawn fixed worker pool
	var wg sync.WaitGroup
	for i := 0; i < opts.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, workQueue, results, s, compiledPattern, opts)
		}()
	}

	// producer goroutine
	go func() {
		defer close(workQueue)

		for _, batch := range batches {
			select {
			case workQueue <- batch:
			case <-ctx.Done():
				return
			}
		}
	}()

	// collector goroutine
	go func() {
		wg.Wait()        // wait for all workers to finish
		close(results)   // signal consumer: no more results

		// record final stats
		duration := time.Since(startTime)
		atomic.StoreInt64(&s.stats.searchDuration, int64(duration))
	}()

	return results, nil
}

// Stats returns current search statistics
func (s *HARSearcher) Stats() SearchStats {
	return SearchStats{
		EntriesSearched: atomic.LoadInt64(&s.stats.entriesSearched),
		MatchesFound:    atomic.LoadInt64(&s.stats.matchesFound),
		BytesSearched:   atomic.LoadInt64(&s.stats.bytesSearched),
		SearchDuration:  time.Duration(atomic.LoadInt64(&s.stats.searchDuration)),
	}
}
