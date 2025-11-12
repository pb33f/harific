package motor

import (
	"context"
	"testing"
)

// integration test to exercise the full pipeline
func TestFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// create streamer
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 2

	streamer, err := NewHARStreamer("../testdata/test-5MB.har", opts)
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()

	// initialize
	if err := streamer.Initialize(ctx); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	// get index
	index := streamer.GetIndex()
	if index == nil {
		t.Fatal("expected non-nil index")
	}

	t.Logf("loaded har: %d entries, %d MB",
		index.TotalEntries,
		index.FileSize/(1024*1024))

	// get single entry
	entry, err := streamer.GetEntry(ctx, 0)
	if err != nil {
		t.Fatalf("get entry failed: %v", err)
	}

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}

	t.Logf("first entry: %s %s -> %d",
		entry.Request.Method,
		entry.Request.URL,
		entry.Response.StatusCode)

	// get metadata
	metadata, err := streamer.GetMetadata(0)
	if err != nil {
		t.Fatalf("get metadata failed: %v", err)
	}

	if metadata.Method != entry.Request.Method {
		t.Errorf("metadata method mismatch: %s != %s",
			metadata.Method, entry.Request.Method)
	}

	// stream range
	count := 5
	if index.TotalEntries < count {
		count = index.TotalEntries
	}

	resultChan, err := streamer.StreamRange(ctx, 0, count)
	if err != nil {
		t.Fatalf("stream range failed: %v", err)
	}

	streamed := 0
	for result := range resultChan {
		if result.Error == nil {
			streamed++
		}
	}

	if streamed == 0 {
		t.Error("expected at least one streamed entry")
	}

	t.Logf("streamed %d entries", streamed)

	// get stats
	stats := streamer.Stats()
	if stats.TotalReads == 0 {
		t.Error("expected non-zero total reads")
	}

	t.Logf("stats: %d reads, %d parsed, avg: %v",
		stats.TotalReads,
		stats.EntriesParsed,
		stats.AverageReadTime)
}
