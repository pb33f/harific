package motor

import (
	"os"
	"testing"
)

func TestNewEntryReader(t *testing.T) {
	// build index first
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	// create reader
	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
}

func TestEntryReader_ReadAt(t *testing.T) {
	// build index
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	// create reader
	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	// read first entry using its byte offset from metadata
	meta := index.Entries[0]
	entry, err := reader.ReadAt(meta.FileOffset, meta.Length)
	if err != nil {
		t.Fatalf("failed to read entry: %v", err)
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

	t.Logf("read entry: %s %s", entry.Request.Method, entry.Request.URL)
}

func TestEntryReader_ReadAtMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple read test in short mode")
	}

	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	count := 5
	if index.TotalEntries < count {
		count = index.TotalEntries
	}

	for i := 0; i < count; i++ {
		meta := index.Entries[i]
		entry, err := reader.ReadAt(meta.FileOffset, meta.Length)
		if err != nil {
			t.Errorf("failed to read entry %d: %v", i, err)
			continue
		}

		if entry == nil {
			t.Errorf("expected non-nil entry for index %d", i)
		}
	}
}

func TestEntryReader_ReadMetadata(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	// read metadata using actual byte offset
	firstOffset := index.Entries[0].FileOffset
	metadata, err := reader.ReadMetadata(firstOffset)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
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
}

func TestEntryReader_ReadMetadataOutOfBounds(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	// try to read out of bounds
	_, err = reader.ReadMetadata(999999)
	if err == nil {
		t.Error("expected error for out of bounds index")
	}

	// try negative index
	_, err = reader.ReadMetadata(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestEntryReader_ReadPartial(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	fields := []string{"request", "response"}
	meta := index.Entries[0]
	partial, err := reader.ReadPartial(meta.FileOffset, fields)
	if err != nil {
		t.Fatalf("failed to read partial: %v", err)
	}

	if partial == nil {
		t.Fatal("expected non-nil partial result")
	}

	if _, ok := partial["request"]; !ok {
		t.Error("expected request field in partial result")
	}

	if _, ok := partial["response"]; !ok {
		t.Error("expected response field in partial result")
	}

	t.Logf("read %d partial fields", len(partial))
}

func TestEntryReader_StreamResponseBody(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	meta := index.Entries[0]
	bodyReader, err := reader.StreamResponseBody(meta.FileOffset)
	if err != nil {
		t.Fatalf("failed to stream body: %v", err)
	}

	if bodyReader == nil {
		t.Fatal("expected non-nil body reader")
	}

	bodyReader.Close()
}

func TestEntryReader_Close(t *testing.T) {
	file, err := os.Open("../testdata/test-5MB.har")
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}

	builder := NewIndexBuilder("../testdata/test-5MB.har")
	index, err := builder.Build(file)
	file.Close()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	reader, err := NewEntryReader("../testdata/test-5MB.har", index)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}

	err = reader.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	// second close should be safe
	err = reader.Close()
	if err != nil {
		t.Errorf("second close failed: %v", err)
	}
}
