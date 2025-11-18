package motor

import (
	"context"
	"os"
	"testing"

	"github.com/pb33f/braid/hargen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_EndToEnd_SmallHAR(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  50,
		InjectTerms: []string{"apple", "banana", "cherry", "date", "elderberry"},
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

	// create streamer
	streamer, err := NewHARStreamer(result.HARFilePath, DefaultStreamerOptions())
	require.NoError(t, err)
	defer streamer.Close()

	err = streamer.Initialize(context.Background())
	require.NoError(t, err)

	// create reader
	index := streamer.GetIndex()
	reader, err := NewEntryReader(result.HARFilePath, index)
	require.NoError(t, err)
	defer reader.Close()

	// create searcher
	searcher := NewSearcher(streamer, reader)

	// search for each injected term
	for _, term := range []string{"apple", "banana", "cherry", "date", "elderberry"} {
		opts := DefaultSearchOptions
		opts.SearchResponseBody = true

		resultChan, err := searcher.Search(context.Background(), term, opts)
		require.NoError(t, err)

		results := collectResults(resultChan)
		assert.GreaterOrEqual(t, len(results), 1, "should find term: %s", term)

		// verify the term was found in expected entry
		foundInExpectedEntry := false
		for _, injected := range result.InjectedTerms {
			if injected.Term == term {
				for _, res := range results {
					if res.Index == injected.EntryIndex {
						foundInExpectedEntry = true
						break
					}
				}
			}
		}
		assert.True(t, foundInExpectedEntry, "term %s should be found in expected entry", term)
	}
}

func TestIntegration_EndToEnd_LargeHAR(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large har test in short mode")
	}

	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  1000,
		InjectTerms: generateTerms(20), // 20 unique terms
		InjectionLocations: []hargen.InjectionLocation{
			hargen.URL,
			hargen.RequestBody,
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

	opts := DefaultSearchOptions
	opts.SearchResponseBody = true
	opts.WorkerCount = 8

	// search with pattern that matches multiple terms
	resultChan, err := searcher.Search(context.Background(), "testterm", opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 10, "should find multiple matches in large har")

	stats := searcher.Stats()
	assert.Equal(t, int64(1000), stats.EntriesSearched)
	assert.Greater(t, stats.BytesSearched, int64(0))
}

func TestIntegration_RegexSearch_MultiplePatterns(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  100,
		InjectTerms: []string{"api001", "api002", "api003", "api999"},
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

	opts := DefaultSearchOptions
	opts.Mode = Regex

	resultChan, err := searcher.Search(context.Background(), `api\d{3}`, opts)
	require.NoError(t, err)

	results := collectResults(resultChan)
	assert.GreaterOrEqual(t, len(results), 3, "should find at least 3 api[3-digit] patterns")
	assert.LessOrEqual(t, len(results), 4, "should find at most 4 api[3-digit] patterns")
}

func TestIntegration_SearchAllFieldTypes(t *testing.T) {
	// create har with terms in each location type
	locationTests := []struct {
		location      hargen.InjectionLocation
		term          string
		needDeepSearch bool
	}{
		{hargen.URL, "urltest", false},
		{hargen.RequestHeader, "reqheadertest", false},
		{hargen.ResponseHeader, "respheadertest", false},
		{hargen.RequestBody, "reqbodytest", false},
		{hargen.ResponseBody, "respbodytest", true},
		{hargen.QueryParam, "querytest", false},
		{hargen.Cookie, "cookietest", false},
	}

	for _, tt := range locationTests {
		t.Run(tt.location.String(), func(t *testing.T) {
			result, err := hargen.Generate(hargen.GenerateOptions{
				EntryCount:  10,
				InjectTerms: []string{tt.term},
				InjectionLocations: []hargen.InjectionLocation{
					tt.location,
				},
				Seed: 42 + int64(tt.location),
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
			if tt.needDeepSearch {
				opts.SearchResponseBody = true
			}

			resultChan, err := searcher.Search(context.Background(), tt.term, opts)
			require.NoError(t, err)

			results := collectResults(resultChan)
			assert.Len(t, results, 1, "should find term in %s", tt.location)
		})
	}
}

// helper function to generate unique test terms
func generateTerms(count int) []string {
	terms := make([]string, count)
	for i := 0; i < count; i++ {
		terms[i] = "testterm" + string(rune(i+'0'))
	}
	return terms
}
