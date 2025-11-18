package motor

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/pb33f/braid/hargen"
	"github.com/pb33f/harhar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorkBatches_AutoPartition(t *testing.T) {
	opts := SearchOptions{
		WorkerCount: 10,
		ChunkSize:   0, // auto-partition
	}

	tests := []struct {
		name           string
		totalEntries   int
		expectedBatches int
		expectedChunkSize int
	}{
		{"1000 entries, 10 workers", 1000, 10, 100},
		{"500 entries, 10 workers", 500, 10, 50},
		{"5000 entries, 10 workers", 5000, 10, 500},
		{"15 entries, 10 workers", 15, 8, 2}, // uneven: 15/10=2 chunks, creates 8 batches
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := createWorkBatches(tt.totalEntries, opts)
			assert.Len(t, batches, tt.expectedBatches)

			// verify coverage
			totalCovered := 0
			for _, batch := range batches {
				totalCovered += batch.endIndex - batch.startIndex
			}
			assert.Equal(t, tt.totalEntries, totalCovered)

			// verify contiguous
			for i := 1; i < len(batches); i++ {
				assert.Equal(t, batches[i-1].endIndex, batches[i].startIndex)
			}
		})
	}
}

func TestCreateWorkBatches_ManualChunkSize(t *testing.T) {
	tests := []struct {
		name             string
		totalEntries     int
		chunkSize        int
		expectedBatches  int
	}{
		{"1000 entries, chunk 250", 1000, 250, 4},
		{"1000 entries, chunk 300", 1000, 300, 4}, // last batch smaller
		{"500 entries, chunk 100", 500, 100, 5},
		{"100 entries, chunk 1000", 100, 1000, 1}, // chunk bigger than total
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SearchOptions{
				WorkerCount: runtime.NumCPU(),
				ChunkSize:   tt.chunkSize,
			}

			batches := createWorkBatches(tt.totalEntries, opts)
			assert.Len(t, batches, tt.expectedBatches)

			totalCovered := 0
			for _, batch := range batches {
				totalCovered += batch.endIndex - batch.startIndex
			}
			assert.Equal(t, tt.totalEntries, totalCovered)
		})
	}
}

func TestSearchMetadata_AllFields(t *testing.T) {
	opts := SearchOptions{Mode: PlainText}

	tests := []struct {
		name         string
		fieldToMatch string
		metadataFunc func(*EntryMetadata, string) // function to set the field
		expectedField string
	}{
		{
			name:         "url match",
			fieldToMatch: "uniqueurl",
			metadataFunc: func(m *EntryMetadata, val string) { m.URL = "https://api.com/" + val },
			expectedField: "url",
		},
		{
			name:         "method match",
			fieldToMatch: "CUSTOMMETHOD",
			metadataFunc: func(m *EntryMetadata, val string) { m.Method = val },
			expectedField: "method",
		},
		{
			name:         "status match",
			fieldToMatch: "Created",
			metadataFunc: func(m *EntryMetadata, val string) { m.StatusText = val },
			expectedField: "status",
		},
		{
			name:         "mimeType match",
			fieldToMatch: "application/customtype",
			metadataFunc: func(m *EntryMetadata, val string) { m.MimeType = val },
			expectedField: "mimeType",
		},
		{
			name:         "serverIP match",
			fieldToMatch: "192.168.1.100",
			metadataFunc: func(m *EntryMetadata, val string) { m.ServerIP = val },
			expectedField: "serverIP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &EntryMetadata{}
			tt.metadataFunc(metadata, tt.fieldToMatch)

			pattern, err := compilePattern(tt.fieldToMatch, opts)
			require.NoError(t, err)

			result := searchMetadata(0, metadata, pattern)
			require.NotNil(t, result)
			assert.Equal(t, 0, result.Index)
			assert.Equal(t, tt.expectedField, result.Field)
			assert.Nil(t, result.Error)
		})
	}
}

func TestSearchMetadata_NoMatch(t *testing.T) {
	metadata := &EntryMetadata{
		URL:        "https://example.com",
		Method:     "GET",
		StatusText: "OK",
		MimeType:   "application/json",
		ServerIP:   "1.2.3.4",
	}

	opts := SearchOptions{Mode: PlainText}
	pattern, err := compilePattern("notfound", opts)
	require.NoError(t, err)

	result := searchMetadata(0, metadata, pattern)
	assert.Nil(t, result)
}

func TestSearchHeaders_RequestHeaders(t *testing.T) {
	headers := []harhar.NameValuePair{
		{Name: "Content-Type", Value: "application/json"},
		{Name: "Authorization", Value: "Bearer token123"},
		{Name: "X-Custom-Header", Value: "custom-value"},
	}

	opts := SearchOptions{Mode: PlainText}

	tests := []struct {
		name          string
		pattern       string
		expectedField string
	}{
		{"match header name", "Authorization", "request.headers.Authorization"},
		{"match header value", "token123", "request.headers.Authorization"},
		{"match custom header", "custom-value", "request.headers.X-Custom-Header"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := compilePattern(tt.pattern, opts)
			require.NoError(t, err)

			result := searchHeaders(0, headers, pattern, "request.headers.")
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedField, result.Field)
		})
	}
}

func TestSearchHeaders_NoMatch(t *testing.T) {
	headers := []harhar.NameValuePair{
		{Name: "Content-Type", Value: "application/json"},
	}

	opts := SearchOptions{Mode: PlainText}
	pattern, err := compilePattern("notfound", opts)
	require.NoError(t, err)

	result := searchHeaders(0, headers, pattern, "request.headers.")
	assert.Nil(t, result)
}

func TestSearchEntry_EarlyReturn_MetadataMatch(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  10,
		InjectTerms: []string{"findmefast"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "findmefast" {
			injectedEntry = &result.InjectedTerms[i]
			break
		}
	}
	require.NotNil(t, injectedEntry)

	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	opts := DefaultSearchOptions
	pattern, err := compilePattern("findmefast", opts)
	require.NoError(t, err)

	buf := make([]byte, 64*1024)
	searchResults := searchEntry(context.Background(), searcher, injectedEntry.EntryIndex, pattern, opts, &buf)

	require.NotEmpty(t, searchResults)
	assert.Equal(t, injectedEntry.EntryIndex, searchResults[0].Index)
	assert.Equal(t, "url", searchResults[0].Field)

	// verify early return: no bytes should have been read from disk
	stats := searcher.Stats()
	assert.Equal(t, int64(0), stats.BytesSearched, "early return should not load entry from disk")
}

func TestSearchEntry_LazyLoading_BodyMatch(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  5,
		InjectTerms: []string{"bodyterm"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.RequestBody,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "bodyterm" {
			injectedEntry = &result.InjectedTerms[i]
			break
		}
	}
	require.NotNil(t, injectedEntry)

	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	opts := DefaultSearchOptions
	pattern, err := compilePattern("bodyterm", opts)
	require.NoError(t, err)

	buf := make([]byte, 64*1024)
	searchResults := searchEntry(context.Background(), searcher, injectedEntry.EntryIndex, pattern, opts, &buf)

	require.NotEmpty(t, searchResults)
	assert.Equal(t, injectedEntry.EntryIndex, searchResults[0].Index)
	assert.Equal(t, "request.body", searchResults[0].Field)

	// verify entry WAS loaded (bytes read > 0)
	stats := searcher.Stats()
	assert.Greater(t, stats.BytesSearched, int64(0), "body match requires loading entry from disk")
}

func TestSearchEntry_AllFieldTypes(t *testing.T) {
	tests := []struct {
		name           string
		injectionLoc   hargen.InjectionLocation
		term           string
		expectedField  string
	}{
		{"url", hargen.URL, "urlterm", "url"},
		{"request header", hargen.RequestHeader, "reqheaderterm", "request.headers."},
		{"response header", hargen.ResponseHeader, "respheaderterm", "response.headers."},
		{"request body", hargen.RequestBody, "reqbodyterm", "request.body"},
		{"response body", hargen.ResponseBody, "respbodyterm", "response.body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hargen.Generate(hargen.GenerateOptions{
				EntryCount:  5,
				InjectTerms: []string{tt.term},
				InjectionLocations: []hargen.InjectionLocation{
					tt.injectionLoc,
				},
				Seed: 42 + int64(tt.injectionLoc), // vary seed per test
			})
			require.NoError(t, err)
			defer os.Remove(result.HARFilePath)

			var injectedEntry *hargen.InjectedTerm
			for i := range result.InjectedTerms {
				if result.InjectedTerms[i].Term == tt.term {
					injectedEntry = &result.InjectedTerms[i]
					break
				}
			}
			require.NotNil(t, injectedEntry)

			streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
			require.NoError(t, err)
			defer streamer.Close()

			err = streamer.Initialize(context.Background())
			require.NoError(t, err)

			index := streamer.GetIndex()
			reader, err := NewEntryReader(result.HARFilePath, index)
			require.NoError(t, err)
			defer reader.Close()

			searcher := NewSearcher(streamer, reader)

			opts := DefaultSearchOptions
			if tt.injectionLoc == hargen.ResponseBody {
				opts.SearchResponseBody = true
			}

			pattern, err := compilePattern(tt.term, opts)
			require.NoError(t, err)

			buf := make([]byte, 64*1024)
			searchResults := searchEntry(context.Background(), searcher, injectedEntry.EntryIndex, pattern, opts, &buf)

			require.NotEmpty(t, searchResults, "should find term in %s", tt.injectionLoc)
			assert.Equal(t, injectedEntry.EntryIndex, searchResults[0].Index)
			assert.Contains(t, searchResults[0].Field, tt.expectedField)
		})
	}
}

func TestSearchEntry_DeepSearchDisabled_SkipsResponseBody(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  3,
		InjectTerms: []string{"deepterm"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.ResponseBody,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "deepterm" {
			injectedEntry = &result.InjectedTerms[i]
			break
		}
	}
	require.NotNil(t, injectedEntry)

	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	opts := DefaultSearchOptions
	opts.SearchResponseBody = false // disabled

	pattern, err := compilePattern("deepterm", opts)
	require.NoError(t, err)

	buf := make([]byte, 64*1024)
	searchResults := searchEntry(context.Background(), searcher, injectedEntry.EntryIndex, pattern, opts, &buf)

	assert.Empty(t, searchResults, "should NOT find term when deep search disabled")
}

func TestSearchEntry_DeepSearchEnabled_FindsResponseBody(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  3,
		InjectTerms: []string{"deepterm2"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.ResponseBody,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "deepterm2" {
			injectedEntry = &result.InjectedTerms[i]
			break
		}
	}
	require.NotNil(t, injectedEntry)

	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	opts := DefaultSearchOptions
	opts.SearchResponseBody = true // enabled

	pattern, err := compilePattern("deepterm2", opts)
	require.NoError(t, err)

	buf := make([]byte, 64*1024)
	searchResults := searchEntry(context.Background(), searcher, injectedEntry.EntryIndex, pattern, opts, &buf)

	require.NotEmpty(t, searchResults)
	assert.Equal(t, "response.body", searchResults[0].Field)
}

func TestSearchEntry_NoMatch(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 5,
		Seed:       42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	opts := DefaultSearchOptions
	pattern, err := compilePattern("definitelyn otfound12345", opts)
	require.NoError(t, err)

	buf := make([]byte, 64*1024)
	searchResults := searchEntry(context.Background(), searcher, 0, pattern, opts, &buf)

	assert.Empty(t, searchResults)
}
