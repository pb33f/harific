package motor

import (
	"context"
	"os"
	"testing"

	"github.com/pb33f/braid/hargen"
	"github.com/stretchr/testify/require"
)

// benchmark configurations for small, medium, and large har files
type benchConfig struct {
	name               string
	entryCount         int
	injectTerms        []string
	injectionLocations []hargen.InjectionLocation
	searchTerm         string
	searchOptions      SearchOptions
	skipInShortMode    bool
}

// runSearchBenchmark executes a search benchmark with given configuration
func runSearchBenchmark(b *testing.B, cfg benchConfig) {
	if cfg.skipInShortMode && testing.Short() {
		b.Skip("skipping large benchmark in short mode")
	}

	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:         cfg.entryCount,
		InjectTerms:        cfg.injectTerms,
		InjectionLocations: cfg.injectionLocations,
		Seed:               42,
	})
	require.NoError(b, err)
	defer os.Remove(result.HARFilePath)

	streamer, reader, searcher := setupSearcher(b, result.HARFilePath)
	defer streamer.Close()
	defer reader.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resultChan, err := searcher.Search(context.Background(), cfg.searchTerm, cfg.searchOptions)
		if err != nil {
			b.Fatal(err)
		}
		drainResults(resultChan)
	}
}

// plain text search benchmarks

func BenchmarkSearch_Small_PlainText(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         100,
		injectTerms:        []string{"searchterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.RequestBody},
		searchTerm:         "searchterm",
		searchOptions:      DefaultSearchOptions,
	})
}

func BenchmarkSearch_Medium_PlainText(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         1000,
		injectTerms:        []string{"searchterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.RequestBody},
		searchTerm:         "searchterm",
		searchOptions:      DefaultSearchOptions,
	})
}

func BenchmarkSearch_Large_PlainText(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         10000,
		injectTerms:        []string{"searchterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.RequestBody},
		searchTerm:         "searchterm",
		searchOptions:      DefaultSearchOptions,
		skipInShortMode:    true,
	})
}

// regex search benchmarks

func BenchmarkSearch_Small_Regex(b *testing.B) {
	opts := DefaultSearchOptions
	opts.Mode = Regex

	runSearchBenchmark(b, benchConfig{
		entryCount:         100,
		injectTerms:        []string{"api123", "api456"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         `api\d+`,
		searchOptions:      opts,
	})
}

func BenchmarkSearch_Medium_Regex(b *testing.B) {
	opts := DefaultSearchOptions
	opts.Mode = Regex

	runSearchBenchmark(b, benchConfig{
		entryCount:         1000,
		injectTerms:        []string{"api123", "api456", "api789"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         `api\d+`,
		searchOptions:      opts,
	})
}

func BenchmarkSearch_Large_Regex(b *testing.B) {
	opts := DefaultSearchOptions
	opts.Mode = Regex

	runSearchBenchmark(b, benchConfig{
		entryCount:         10000,
		injectTerms:        []string{"api123", "api456", "api789"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         `api\d+`,
		searchOptions:      opts,
		skipInShortMode:    true,
	})
}

// metadata-only search benchmarks (early return optimization)

func BenchmarkSearch_Small_MetadataOnly(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         100,
		injectTerms:        []string{"metafast"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         "metafast",
		searchOptions:      DefaultSearchOptions,
	})
}

func BenchmarkSearch_Medium_MetadataOnly(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         1000,
		injectTerms:        []string{"metafast"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         "metafast",
		searchOptions:      DefaultSearchOptions,
	})
}

func BenchmarkSearch_Large_MetadataOnly(b *testing.B) {
	runSearchBenchmark(b, benchConfig{
		entryCount:         10000,
		injectTerms:        []string{"metafast"},
		injectionLocations: []hargen.InjectionLocation{hargen.URL},
		searchTerm:         "metafast",
		searchOptions:      DefaultSearchOptions,
		skipInShortMode:    true,
	})
}

// deep search benchmarks (response bodies)

func BenchmarkSearch_Small_DeepSearch(b *testing.B) {
	opts := DefaultSearchOptions
	opts.SearchResponseBody = true

	runSearchBenchmark(b, benchConfig{
		entryCount:         100,
		injectTerms:        []string{"deepterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.ResponseBody},
		searchTerm:         "deepterm",
		searchOptions:      opts,
	})
}

func BenchmarkSearch_Medium_DeepSearch(b *testing.B) {
	opts := DefaultSearchOptions
	opts.SearchResponseBody = true

	runSearchBenchmark(b, benchConfig{
		entryCount:         1000,
		injectTerms:        []string{"deepterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.ResponseBody},
		searchTerm:         "deepterm",
		searchOptions:      opts,
	})
}

func BenchmarkSearch_Large_DeepSearch(b *testing.B) {
	opts := DefaultSearchOptions
	opts.SearchResponseBody = true

	runSearchBenchmark(b, benchConfig{
		entryCount:         10000,
		injectTerms:        []string{"deepterm"},
		injectionLocations: []hargen.InjectionLocation{hargen.ResponseBody},
		searchTerm:         "deepterm",
		searchOptions:      opts,
		skipInShortMode:    true,
	})
}

// worker parallelism benchmark

func BenchmarkSearch_ParallelWorkers(b *testing.B) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:         1000,
		InjectTerms:        []string{"parallel"},
		InjectionLocations: []hargen.InjectionLocation{hargen.RequestBody},
		Seed:               42,
	})
	require.NoError(b, err)
	defer os.Remove(result.HARFilePath)

	streamer, reader, searcher := setupSearcher(b, result.HARFilePath)
	defer streamer.Close()
	defer reader.Close()

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(itoa(workers)+"_workers", func(b *testing.B) {
			opts := DefaultSearchOptions
			opts.WorkerCount = workers

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				resultChan, err := searcher.Search(context.Background(), "parallel", opts)
				if err != nil {
					b.Fatal(err)
				}
				drainResults(resultChan)
			}
		})
	}
}

// helper functions

// creates streamer, reader, and searcher for benchmarking
func setupSearcher(b *testing.B, harFilePath string) (HARStreamer, EntryReader, *HARSearcher) {
	streamer, err := NewHARStreamer(harFilePath, DefaultStreamerOptions())
	require.NoError(b, err)

	err = streamer.Initialize(context.Background())
	require.NoError(b, err)

	index := streamer.GetIndex()
	reader, err := NewEntryReader(harFilePath, index)
	require.NoError(b, err)

	searcher := NewSearcher(streamer, reader)

	return streamer, reader, searcher
}

// drainResults drains the result channel without allocating memory for collection
func drainResults(ch <-chan []SearchResult) int {
	count := 0
	for batch := range ch {
		count += len(batch)
	}
	return count
}

// itoa converts int to string for benchmark names
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
