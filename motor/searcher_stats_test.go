package motor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearcher_StatsResetBetweenSearches tests that statistics are reset for each new search
func TestSearcher_StatsResetBetweenSearches(t *testing.T) {
	// Create streamer and initialize
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", DefaultStreamerOptions())
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)
	defer streamer.Close()

	// Create searcher
	reader, err := NewEntryReader("../testdata/test-5MB.har", streamer.GetIndex())
	require.Nil(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	// First search
	opts1 := DefaultSearchOptions
	opts1.WorkerCount = 2
	resultsChan1, err := searcher.Search(context.Background(), "GET", opts1)
	require.Nil(t, err)

	// Consume all results from first search
	matchCount1 := 0
	for results := range resultsChan1 {
		matchCount1 += len(results)
	}

	// Get stats after first search
	stats1 := searcher.Stats()
	assert.Greater(t, stats1.EntriesSearched, int64(0), "should have searched some entries")
	assert.Greater(t, stats1.MatchesFound, int64(0), "should have found some matches")
	firstSearchEntries := stats1.EntriesSearched
	firstSearchMatches := stats1.MatchesFound

	// Second search with different pattern
	opts2 := DefaultSearchOptions
	opts2.WorkerCount = 2
	resultsChan2, err := searcher.Search(context.Background(), "POST", opts2)
	require.Nil(t, err)

	// Consume all results from second search
	matchCount2 := 0
	for results := range resultsChan2 {
		matchCount2 += len(results)
	}

	// Get stats after second search
	stats2 := searcher.Stats()

	// Stats should be reset, not cumulative
	// The second search should have its own counts, not added to the first
	assert.LessOrEqual(t, stats2.EntriesSearched, firstSearchEntries+10,
		"stats should be reset, not cumulative from previous search. Got %d, expected around %d or less",
		stats2.EntriesSearched, firstSearchEntries)

	// Verify stats reflect only the second search
	assert.Greater(t, stats2.EntriesSearched, int64(0), "should have searched entries in second search")

	// The matches from second search should be different from first
	// (unless by coincidence they match the same count)
	t.Logf("First search: %d entries, %d matches", firstSearchEntries, firstSearchMatches)
	t.Logf("Second search: %d entries, %d matches", stats2.EntriesSearched, stats2.MatchesFound)
}

// TestSearcher_MultipleSequentialSearches tests multiple searches on same searcher instance
func TestSearcher_MultipleSequentialSearches(t *testing.T) {
	// Create and initialize streamer
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", DefaultStreamerOptions())
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)
	defer streamer.Close()

	// Create searcher
	reader, err := NewEntryReader("../testdata/test-5MB.har", streamer.GetIndex())
	require.Nil(t, err)
	defer reader.Close()

	searcher := NewSearcher(streamer, reader)

	patterns := []string{"GET", "POST", "api", "json", "http"}

	for i, pattern := range patterns {
		t.Logf("Search %d: pattern='%s'", i+1, pattern)

		opts := DefaultSearchOptions
		opts.WorkerCount = 2

		resultsChan, err := searcher.Search(context.Background(), pattern, opts)
		require.Nil(t, err, "search %d should not error", i+1)

		// Consume results
		matchCount := 0
		for results := range resultsChan {
			matchCount += len(results)
		}

		// Get stats
		stats := searcher.Stats()

		t.Logf("  Entries searched: %d", stats.EntriesSearched)
		t.Logf("  Matches found: %d", stats.MatchesFound)
		t.Logf("  Duration: %v", stats.SearchDuration)

		// Each search should have reasonable stats (not accumulated from previous searches)
		assert.Greater(t, stats.EntriesSearched, int64(0), "search %d should have searched entries", i+1)
		assert.GreaterOrEqual(t, stats.MatchesFound, int64(0), "search %d should have valid match count", i+1)

		// Stats should be bounded by total entries in file (~67 entries in test-5MB.har)
		assert.LessOrEqual(t, stats.EntriesSearched, int64(100),
			"search %d stats should not be cumulative (got %d entries)", i+1, stats.EntriesSearched)
	}
}
