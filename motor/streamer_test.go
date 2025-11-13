package motor

import (
	"context"
	"testing"
	"time"
)

func TestNewHARStreamer(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}

	if streamer == nil {
		t.Fatal("expected non-nil streamer")
	}

	if streamer.filePath != "../testdata/test-5MB.har" {
		t.Errorf("expected filepath ../testdata/test-5MB.har, got %s", streamer.filePath)
	}
}

func TestHARStreamer_Initialize(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}

	ctx := context.Background()
	err = streamer.Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// verify index was built
	if streamer.index == nil {
		t.Fatal("expected non-nil index")
	}

	if streamer.index.TotalEntries <= 0 {
		t.Error("expected at least one entry")
	}

	// cleanup
	streamer.Close()
}

func TestHARStreamer_GetEntry(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// get first entry
	entry, err := streamer.GetEntry(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	if entry.Request.Method == "" {
		t.Error("expected non-empty method")
	}

	if entry.Request.URL == "" {
		t.Error("expected non-empty url")
	}

	t.Logf("entry 0: %s %s", entry.Request.Method, entry.Request.URL)
}

func TestHARStreamer_GetEntryOutOfBounds(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// try to get entry beyond bounds
	_, err = streamer.GetEntry(ctx, 999999)
	if err == nil {
		t.Error("expected error for out of bounds index")
	}

	// try negative index
	_, err = streamer.GetEntry(ctx, -1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestHARStreamer_GetMetadata(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// get metadata for first entry
	metadata, err := streamer.GetMetadata(0)
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if metadata == nil {
		t.Fatal("expected non-nil metadata")
	}

	if metadata.Method == "" {
		t.Error("expected non-empty method")
	}

	if metadata.URL == "" {
		t.Error("expected non-empty url")
	}

	t.Logf("metadata 0: %s %s -> %d", metadata.Method, metadata.URL, metadata.StatusCode)
}

func TestHARStreamer_StreamRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stream test in short mode")
	}

	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// stream first 10 entries
	end := 10
	if streamer.index.TotalEntries < end {
		end = streamer.index.TotalEntries
	}

	resultChan, err := streamer.StreamRange(ctx, 0, end)
	if err != nil {
		t.Fatalf("failed to stream range: %v", err)
	}

	// collect results
	count := 0
	errors := 0
	for result := range resultChan {
		if result.Error != nil {
			errors++
			t.Logf("error reading entry %d: %v", result.Index, result.Error)
		} else {
			count++
		}
	}

	if count == 0 {
		t.Error("expected at least one successful result")
	}

	t.Logf("streamed %d entries with %d errors", count, errors)
}

func TestHARStreamer_StreamFiltered(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filtered stream test in short mode")
	}

	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// filter for GET requests
	filter := func(meta *EntryMetadata) bool {
		return meta.Method == "GET"
	}

	resultChan, err := streamer.StreamFiltered(ctx, filter)
	if err != nil {
		t.Fatalf("failed to stream filtered: %v", err)
	}

	// collect results
	count := 0
	for result := range resultChan {
		if result.Error == nil {
			count++
			if result.Entry.Request.Method != "GET" {
				t.Errorf("expected GET method, got %s", result.Entry.Request.Method)
			}
		}
	}

	t.Logf("filtered %d GET requests", count)
}

func TestHARStreamer_StreamRangeWithCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cancel test in short mode")
	}

	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-50MB.har", opts)
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// stream all entries but cancel after a few
	resultChan, err := streamer.StreamRange(ctx, 0, streamer.index.TotalEntries)
	if err != nil {
		t.Fatalf("failed to stream range: %v", err)
	}

	count := 0
	for result := range resultChan {
		if result.Error == nil {
			count++
			if count >= 5 {
				cancel()
			}
		}
	}

	if count == 0 {
		t.Error("expected at least one result before cancel")
	}

	t.Logf("processed %d entries before cancel", count)
}

func TestHARStreamer_Stats(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// get initial stats
	stats := streamer.Stats()
	if stats.TotalReads != 0 {
		t.Errorf("expected 0 reads initially, got %d", stats.TotalReads)
	}

	// read some entries
	for i := 0; i < 3; i++ {
		_, err := streamer.GetEntry(ctx, i)
		if err != nil {
			t.Logf("failed to get entry %d: %v", i, err)
		}
	}

	// check updated stats
	stats = streamer.Stats()
	if stats.TotalReads == 0 {
		t.Error("expected non-zero reads after getting entries")
	}

	t.Logf("stats: %d reads, %d parsed, avg time: %v",
		stats.TotalReads, stats.EntriesParsed, stats.AverageReadTime)
}

func TestHARStreamer_GetIndex(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	index := streamer.GetIndex()
	if index == nil {
		t.Fatal("expected non-nil index")
	}

	if index.TotalEntries <= 0 {
		t.Error("expected at least one entry in index")
	}
}

func TestHARStreamer_WithTimeout(t *testing.T) {
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Fatalf("failed to create streamer: %v", err)
	}
	defer streamer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// should complete within timeout
	_, err = streamer.GetEntry(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get entry: %v", err)
	}
}
