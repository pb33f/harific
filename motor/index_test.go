package motor

import (
	"os"
	"testing"
	"time"
)

func TestIndexBuilder_Build_5MB(t *testing.T) {
	// open test file
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	// build index
	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	// verify index properties
	if index.FilePath != "../testdata/test-5MB.har" {
		t.Errorf("expected filepath ../testdata/test-5MB.har, got %s", index.FilePath)
	}

	if index.TotalEntries <= 0 {
		t.Error("expected at least one entry")
	}

	if index.FileSize <= 0 {
		t.Error("expected positive file size")
	}

	if index.FileHash == "" {
		t.Error("expected non-empty file hash")
	}

	if len(index.Entries) != index.TotalEntries {
		t.Errorf("expected %d entries, got %d", index.TotalEntries, len(index.Entries))
	}

	if index.BuildTime <= 0 {
		t.Error("expected positive build time")
	}

	t.Logf("built index: %d entries, %d bytes, hash: %s, build time: %v",
		index.TotalEntries, index.FileSize, index.FileHash, index.BuildTime)
}

func TestIndexBuilder_Build_50MB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 50MB test in short mode")
	}

	file, err := os.Open("../testdata/test-50MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	builder := NewIndexBuilder("../testdata/test-50MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	if index.TotalEntries <= 0 {
		t.Error("expected at least one entry")
	}

	t.Logf("built index: %d entries, build time: %v",
		index.TotalEntries, index.BuildTime)
}

func TestIndexBuilder_Metadata(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	// verify first entry metadata
	if len(index.Entries) == 0 {
		t.Fatal("no entries in index")
	}

	first := index.Entries[0]

	if first.Method == "" {
		t.Error("expected non-empty method")
	}

	if first.URL == "" {
		t.Error("expected non-empty url")
	}

	if first.StatusCode == 0 {
		t.Error("expected non-zero status code")
	}

	if first.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	t.Logf("first entry: %s %s -> %d %s",
		first.Method, first.URL, first.StatusCode, first.StatusText)
}

func TestIndexBuilder_TimeRange(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	if index.TimeRange.Start.IsZero() {
		t.Error("expected non-zero start time")
	}

	if index.TimeRange.End.IsZero() {
		t.Error("expected non-zero end time")
	}

	if index.TimeRange.End.Before(index.TimeRange.Start) {
		t.Error("expected end time after start time")
	}

	duration := index.TimeRange.End.Sub(index.TimeRange.Start)
	t.Logf("time range: %v to %v (duration: %v)",
		index.TimeRange.Start, index.TimeRange.End, duration)
}

func TestIndexBuilder_StringInterning(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	// count unique urls
	urlCount := make(map[string]int)
	for _, entry := range index.Entries {
		urlCount[entry.URL]++
	}

	if index.UniqueURLs != len(urlCount) {
		t.Errorf("expected %d unique urls, got %d", len(urlCount), index.UniqueURLs)
	}

	t.Logf("unique urls: %d", index.UniqueURLs)
}

func TestIndexBuilder_TotalBytes(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer file.Close()

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	if index.TotalRequestBytes < 0 {
		t.Error("expected non-negative request bytes")
	}

	if index.TotalResponseBytes < 0 {
		t.Error("expected non-negative response bytes")
	}

	t.Logf("total bytes: %d requests, %d responses",
		index.TotalRequestBytes, index.TotalResponseBytes)
}

func TestIndexBuilder_AddEntry(t *testing.T) {
	builder := NewIndexBuilder("test.har")

	metadata := &EntryMetadata{
		FileOffset: 1024,
		Length:     2048,
		Method:     "GET",
		URL:        "https://test.com",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}

	err := builder.AddEntry(1024, metadata)
	if err != nil {
		t.Errorf("add entry failed: %v", err)
	}

	index := builder.GetIndex()
	if len(index.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(index.Entries))
	}

	if index.Entries[0].FileOffset != 1024 {
		t.Errorf("expected offset 1024, got %d", index.Entries[0].FileOffset)
	}
}
