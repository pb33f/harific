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
				entryResults := searchEntry(ctx, searcher, i, pattern, opts, buf)

				// flatten results from this entry into batch
				for _, result := range entryResults {
					batchResults = append(batchResults, *result)
				}

				atomic.AddInt64(&searcher.stats.entriesSearched, 1)
			}

			// return buffer to pool immediately
			searcher.bufferPool.Put(buf)

			// send batch results if any matches found
			if len(batchResults) > 0 {
				// count only successful matches (exclude errors)
				successCount := 0
				for _, result := range batchResults {
					if result.Error == nil {
						successCount++
					}
				}

				select {
				case results <- batchResults:
					atomic.AddInt64(&searcher.stats.matchesFound, int64(successCount))
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// searchEntry searches a single entry for pattern matches
// returns slice of matches (empty if no matches, or single error result)
func searchEntry(ctx context.Context,
	s *HARSearcher,
	index int,
	pattern compiledPattern,
	opts SearchOptions,
	buf *[]byte) []*SearchResult {

	var results []*SearchResult

	// step 1: metadata search (no i/o - instant lookup)
	metadata, err := s.streamer.GetMetadata(index)
	if err != nil {
		return []*SearchResult{{Index: index, Error: err}}
	}

	// search metadata fields
	if result := searchMetadata(index, metadata, pattern); result != nil {
		results = append(results, result)
		if opts.FirstMatchOnly && !opts.SearchResponseBody {
			return results // early return unless deep search required
		}
	}

	// decide whether to load full entry
	// skip loading if: firstmatchonly=true AND already matched AND no deep search required
	needsFullEntry := len(results) == 0 || !opts.FirstMatchOnly || opts.SearchResponseBody

	if !needsFullEntry {
		return results
	}

	// step 2: load full entry for header/body searches (if needed)
	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		WithBuffer(buf).
		Build()

	resp := s.reader.Read(ctx, req)
	if resp.GetError() != nil {
		return []*SearchResult{{Index: index, Error: resp.GetError()}}
	}

	entry := resp.GetEntry()
	atomic.AddInt64(&s.stats.bytesSearched, resp.GetBytesRead())

	// step 3: search request headers
	if result := searchHeaders(index, entry.Request.Headers, pattern, "request.headers."); result != nil {
		results = append(results, result)
		if opts.FirstMatchOnly && !opts.SearchResponseBody {
			return results
		}
	}

	// step 4: search query params
	if result := searchHeaders(index, entry.Request.QueryParams, pattern, "query.param."); result != nil {
		results = append(results, result)
		if opts.FirstMatchOnly && !opts.SearchResponseBody {
			return results
		}
	}

	// step 5: search cookies
	if result := searchCookies(index, entry.Request.Cookies, pattern); result != nil {
		results = append(results, result)
		if opts.FirstMatchOnly && !opts.SearchResponseBody {
			return results
		}
	}

	// step 6: search request body
	if entry.Request.Body.Content != "" {
		if matches(entry.Request.Body.Content, pattern) {
			results = append(results, &SearchResult{Index: index, Field: "request.body"})
			if opts.FirstMatchOnly && !opts.SearchResponseBody {
				return results
			}
		}
	}

	// step 7: search response headers
	if result := searchHeaders(index, entry.Response.Headers, pattern, "response.headers."); result != nil {
		results = append(results, result)
		if opts.FirstMatchOnly && !opts.SearchResponseBody {
			return results
		}
	}

	// step 8: ALWAYS search response body if deep search enabled (guarantees bodies are checked)
	if opts.SearchResponseBody && entry.Response.Body.Content != "" {
		if matches(entry.Response.Body.Content, pattern) {
			results = append(results, &SearchResult{Index: index, Field: "response.body"})
		}
	}

	return results
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

// searchCookies checks if any cookie name or value matches the pattern
func searchCookies(index int, cookies []harhar.Cookie, pattern compiledPattern) *SearchResult {
	for _, cookie := range cookies {
		if matches(cookie.Name, pattern) || matches(cookie.Value, pattern) {
			return &SearchResult{Index: index, Field: "cookie." + cookie.Name}
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
