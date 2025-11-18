package motor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pb33f/braid/hargen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearcher(t *testing.T) {
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

	require.NotNil(t, searcher)
	require.NotNil(t, searcher.bufferPool)
	require.NotNil(t, searcher.streamer)
	require.NotNil(t, searcher.reader)
}

func TestSearch_EmptyHAR(t *testing.T) {
	// test searching with no matches (simulates empty result set)
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 1,
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

	// search for something that definitely won't match
	resultChan, err := searcher.Search(context.Background(), "xyznotfound999", DefaultSearchOptions)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.Len(t, results, 0)
}

func TestSearch_PlainText_SingleMatch(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  10,
		InjectTerms: []string{"uniqueterm"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "uniqueterm" {
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

	resultChan, err := searcher.Search(context.Background(), "uniqueterm", DefaultSearchOptions)
	require.NoError(t, err)

	results := collectResults(resultChan)
	require.Len(t, results, 1)
	assert.Equal(t, injectedEntry.EntryIndex, results[0].Index)
	assert.Equal(t, "url", results[0].Field)
}

func TestSearch_PlainText_MultipleMatches(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  20,
		InjectTerms: []string{"term1", "term2", "term3"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.RequestBody,
			hargen.ResponseHeader,
			hargen.URL,
		},
		Seed: 42,
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

	resultChan, err := searcher.Search(context.Background(), "term", DefaultSearchOptions)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 3, "should find at least 3 matches (all contain 'term')")
}

func TestSearch_Regex_Pattern(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  15,
		InjectTerms: []string{"api123", "api456", "api789"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
		},
		Seed: 42,
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
	opts.Mode = Regex

	resultChan, err := searcher.Search(context.Background(), `api\d+`, opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 2, "should find at least 2 api[digits] patterns")
	assert.LessOrEqual(t, len(results), 3, "should find at most 3 api[digits] patterns")
}

func TestSearch_InvalidPattern_ReturnsError(t *testing.T) {
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
	opts.Mode = Regex

	_, err = searcher.Search(context.Background(), "[invalid(", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pattern")
}

func TestSearch_ConcurrentWorkers(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  100,
		InjectTerms: []string{"worker1", "worker2", "worker3", "worker4", "worker5"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.RequestBody,
			hargen.ResponseHeader,
		},
		Seed: 42,
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
	opts.WorkerCount = 8

	resultChan, err := searcher.Search(context.Background(), "worker", opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 5, "should find all injected terms")

	stats := searcher.Stats()
	assert.Equal(t, int64(100), stats.EntriesSearched)
	assert.GreaterOrEqual(t, stats.MatchesFound, int64(5))
	assert.Greater(t, stats.SearchDuration, time.Duration(0))
}

func TestSearch_ContextCancellation(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 500,
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

	ctx, cancel := context.WithCancel(context.Background())

	opts := DefaultSearchOptions
	opts.WorkerCount = 4
	opts.ChunkSize = 100

	resultChan, err := searcher.Search(ctx, "anything", opts)
	require.NoError(t, err)

	// cancel immediately
	cancel()

	_ = collectResults(resultChan)

	// search should complete gracefully (may have processed some entries)
	stats := searcher.Stats()
	assert.LessOrEqual(t, stats.EntriesSearched, int64(500), "entries searched should not exceed total")
}

func TestStats_Tracking(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  50,
		InjectTerms: []string{"statterm1", "statterm2"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
			hargen.RequestBody,
		},
		Seed: 42,
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

	resultChan, err := searcher.Search(context.Background(), "statterm", DefaultSearchOptions)
	require.NoError(t, err)

	results := collectResults(resultChan)

	stats := searcher.Stats()
	assert.Equal(t, int64(50), stats.EntriesSearched, "should search all 50 entries")
	assert.Equal(t, int64(len(results)), stats.MatchesFound)
	assert.Greater(t, stats.BytesSearched, int64(0))
	assert.Greater(t, stats.SearchDuration, time.Duration(0))
}

func TestSearch_CustomWorkerCount(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 100,
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
	opts.WorkerCount = 16 // custom worker count

	resultChan, err := searcher.Search(context.Background(), "test", opts)
	require.NoError(t, err)

	_ = collectResults(resultChan)

	stats := searcher.Stats()
	assert.Equal(t, int64(100), stats.EntriesSearched)
}

func TestSearch_CustomChunkSize(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  200,
		InjectTerms: []string{"chunked"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.RequestBody,
		},
		Seed: 42,
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
	opts.ChunkSize = 25 // custom chunk size

	resultChan, err := searcher.Search(context.Background(), "chunked", opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 1)

	stats := searcher.Stats()
	assert.Equal(t, int64(200), stats.EntriesSearched)
}

func TestSearch_DeepSearchEnabled(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  20,
		InjectTerms: []string{"deepbody"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.ResponseBody,
		},
		Seed: 42,
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

	// first search WITHOUT deep search - should not find
	opts := DefaultSearchOptions
	opts.SearchResponseBody = false

	resultChan, err := searcher.Search(context.Background(), "deepbody", opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.Len(t, results, 0, "should not find response body matches when deep search disabled")

	// now search WITH deep search - should find
	opts.SearchResponseBody = true

	resultChan, err = searcher.Search(context.Background(), "deepbody", opts)
	require.NoError(t, err)

	results = collectResults(resultChan)
	assert.Len(t, results, 1, "should find response body match when deep search enabled")
	assert.Equal(t, "response.body", results[0].Field)
}

func TestSearch_NoMatches(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 30,
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

	resultChan, err := searcher.Search(context.Background(), "definitelynotfound12345xyz", DefaultSearchOptions)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.Len(t, results, 0)

	stats := searcher.Stats()
	assert.Equal(t, int64(30), stats.EntriesSearched)
	assert.Equal(t, int64(0), stats.MatchesFound)
}

func TestSearch_DefaultWorkerCount(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 10,
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
	opts.WorkerCount = 0 // should default to runtime.numcpu()

	resultChan, err := searcher.Search(context.Background(), "anything", opts)
	require.NoError(t, err)

	_ = collectResults(resultChan)

	stats := searcher.Stats()
	assert.Equal(t, int64(10), stats.EntriesSearched)
}

func TestSearch_MultipleInjectionLocations(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  50,
		InjectTerms: []string{"multiterm"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
			hargen.RequestBody,
			hargen.ResponseBody,
			hargen.RequestHeader,
			hargen.ResponseHeader,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	injectedCount := len(result.InjectedTerms)
	require.Equal(t, 1, injectedCount, "should inject 1 term")

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
	opts.SearchResponseBody = true

	resultChan, err := searcher.Search(context.Background(), "multiterm", opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.Len(t, results, 1, "should find the single injected term")
}

// collectResults is a helper to collect all search results from a channel
func collectResults(ch <-chan []SearchResult) []SearchResult {
	var all []SearchResult
	for batch := range ch {
		all = append(all, batch...)
	}
	return all
}
