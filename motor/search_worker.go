package motor

import (
	"context"
	"sync/atomic"

	"github.com/pb33f/harhar"
)

// workBatch represents a range of entries to process
type workBatch struct {
	startIndex int // inclusive start
	endIndex   int // exclusive end (go range convention)
}

// createWorkBatches divides entries into batches for workers
func createWorkBatches(totalEntries int, opts SearchOptions) []workBatch {
	var batches []workBatch

	chunkSize := opts.ChunkSize

	// fallback: auto-partition based on worker count
	if chunkSize == 0 {
		chunkSize = (totalEntries + opts.WorkerCount - 1) / opts.WorkerCount
	}

	// create batches
	for start := 0; start < totalEntries; start += chunkSize {
		end := start + chunkSize
		if end > totalEntries {
			end = totalEntries
		}

		batches = append(batches, workBatch{
			startIndex: start,
			endIndex:   end,
		})
	}

	return batches
}

// worker processes work batches and searches entries
func worker(ctx context.Context,
	workQueue <-chan workBatch,
	results chan<- []SearchResult,
	searcher *HARSearcher,
	pattern compiledPattern,
	opts SearchOptions) {

	for {
		select {
		case <-ctx.Done():
			return

		case batch, ok := <-workQueue:
			if !ok {
				return // work queue closed, all done
			}

			// get buffer from pool once per batch
			buf := searcher.bufferPool.Get().(*[]byte)

			// accumulate matches (small initial capacity for common case)
			batchResults := make([]SearchResult, 0, 8)

			// process each entry in this batch
			for i := batch.startIndex; i < batch.endIndex; i++ {
				result := searchEntry(ctx, searcher, i, pattern, opts, buf)

				if result != nil {
					batchResults = append(batchResults, *result)
				}

				atomic.AddInt64(&searcher.stats.entriesSearched, 1)
			}

			// return buffer to pool immediately
			searcher.bufferPool.Put(buf)

			// send batch results if any matches found
			if len(batchResults) > 0 {
				select {
				case results <- batchResults:
					atomic.AddInt64(&searcher.stats.matchesFound, int64(len(batchResults)))
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// searchEntry searches a single entry for pattern matches
func searchEntry(ctx context.Context,
	s *HARSearcher,
	index int,
	pattern compiledPattern,
	opts SearchOptions,
	buf *[]byte) *SearchResult {

	// step 1: metadata search (no i/o - instant lookup)
	metadata, err := s.streamer.GetMetadata(index)
	if err != nil {
		return &SearchResult{Index: index, Error: err}
	}

	// search metadata fields with early return to skip expensive entry loading
	if result := searchMetadata(index, metadata, pattern); result != nil {
		return result
	}

	// step 2: only load full entry if metadata didn't match (lazy loading)
	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		WithBuffer(buf). // pooled buffer for efficiency
		Build()

	// step 3: read entry using pooled buffer
	resp := s.reader.Read(ctx, req)
	if resp.GetError() != nil {
		return &SearchResult{Index: index, Error: resp.GetError()}
	}

	entry := resp.GetEntry()
	atomic.AddInt64(&s.stats.bytesSearched, resp.GetBytesRead())

	// step 4: search request headers
	if result := searchHeaders(index, entry.Request.Headers, pattern, "request.headers."); result != nil {
		return result
	}

	// step 5: search request body
	if entry.Request.Body.Content != "" {
		if matches(entry.Request.Body.Content, pattern) {
			return &SearchResult{Index: index, Field: "request.body"}
		}
	}

	// step 6: search response headers
	if result := searchHeaders(index, entry.Response.Headers, pattern, "response.headers."); result != nil {
		return result
	}

	// step 7: search response body (optional - deep search)
	if opts.SearchResponseBody && entry.Response.Body.Content != "" {
		if matches(entry.Response.Body.Content, pattern) {
			return &SearchResult{Index: index, Field: "response.body"}
		}
	}

	return nil // no match
}

// searchHeaders checks if any header name or value matches the pattern
func searchHeaders(index int, headers []harhar.NameValuePair, pattern compiledPattern, prefix string) *SearchResult {
	for _, header := range headers {
		if matches(header.Name, pattern) || matches(header.Value, pattern) {
			return &SearchResult{Index: index, Field: prefix + header.Name}
		}
	}
	return nil
}

// searchMetadata checks if any metadata field matches the pattern
func searchMetadata(index int, metadata *EntryMetadata, pattern compiledPattern) *SearchResult {
	metadataFields := []struct {
		value string
		name  string
	}{
		{metadata.URL, "url"},
		{metadata.Method, "method"},
		{metadata.StatusText, "status"},
		{metadata.MimeType, "mimeType"},
		{metadata.ServerIP, "serverIP"},
	}

	for _, field := range metadataFields {
		if matches(field.value, pattern) {
			return &SearchResult{Index: index, Field: field.name}
		}
	}
	return nil
}
