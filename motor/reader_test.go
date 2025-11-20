package motor

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/pb33f/harific/hargen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// new tests for read() method and file pool

func TestRead_WithBuffer_Success(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount:  5,
		InjectTerms: []string{"findme"},
		InjectionLocations: []hargen.InjectionLocation{
			hargen.ResponseBody,
		},
		Seed: 42,
	})
	require.NoError(t, err)
	defer os.Remove(result.HARFilePath)

	var injectedEntry *hargen.InjectedTerm
	for i := range result.InjectedTerms {
		if result.InjectedTerms[i].Term == "findme" {
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

	metadata := index.Entries[injectedEntry.EntryIndex]

	buf := make([]byte, 64*1024)
	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		WithBuffer(&buf).
		Build()

	resp := reader.Read(context.Background(), req)
	require.Nil(t, resp.GetError())
	require.NotNil(t, resp.GetEntry())
	assert.Greater(t, resp.GetBytesRead(), int64(0))
	assert.Contains(t, resp.GetEntry().Response.Body.Content, "findme")
}

func TestRead_WithoutBuffer_Fallback(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 3,
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

	metadata := index.Entries[0]

	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		Build()

	resp := reader.Read(context.Background(), req)
	require.Nil(t, resp.GetError())
	require.NotNil(t, resp.GetEntry())
}

func TestReadMetadata_OffsetIndex_O1Lookup(t *testing.T) {
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

	for i, expected := range index.Entries {
		meta, err := reader.ReadMetadata(expected.FileOffset)
		require.NoError(t, err, "failed to read metadata for entry %d", i)
		assert.Equal(t, expected, meta)
	}
}

func TestReadMetadata_NotFound_ReturnsError(t *testing.T) {
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

	_, err = reader.ReadMetadata(999999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata not found")
}

func TestFilePool_ConcurrentReads_NoMutexContention(t *testing.T) {
	result, err := hargen.Generate(hargen.GenerateOptions{
		EntryCount: 50,
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

	workerCount := 20
	var wg sync.WaitGroup
	errors := make(chan error, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < 5; j++ {
				entryIdx := (workerID*5 + j) % len(index.Entries)
				metadata := index.Entries[entryIdx]

				buf := make([]byte, 64*1024)
				req := NewReadRequestBuilder().
					WithOffset(metadata.FileOffset).
					WithLength(metadata.Length).
					WithBuffer(&buf).
					Build()

				resp := reader.Read(context.Background(), req)
				if resp.GetError() != nil {
					errors <- resp.GetError()
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent read error: %v", err)
	}
}

func TestFilePool_SyncOnce_NoDuplicateTracking(t *testing.T) {
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

	metadata := index.Entries[0]
	for i := 0; i < 100; i++ {
		buf := make([]byte, 64*1024)
		req := NewReadRequestBuilder().
			WithOffset(metadata.FileOffset).
			WithLength(metadata.Length).
			WithBuffer(&buf).
			Build()

		resp := reader.Read(context.Background(), req)
		require.Nil(t, resp.GetError())
	}

	reader.mu.Lock()
	pooledCount := len(reader.pooledFiles)
	reader.mu.Unlock()

	assert.Less(t, pooledCount, 100, "file pool should not grow unbounded - max one per iteration")
}

func TestRead_BufferTooSmall_GrowsBuffer(t *testing.T) {
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

	metadata := index.Entries[0]

	buf := make([]byte, 10)
	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		WithBuffer(&buf).
		Build()

	resp := reader.Read(context.Background(), req)

	require.Nil(t, resp.GetError())
	require.NotNil(t, resp.GetEntry())
	assert.GreaterOrEqual(t, cap(buf), int(metadata.Length))
}

func TestClose_ClosesAllPooledFiles(t *testing.T) {
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

	for i := 0; i < 3 && i < len(index.Entries); i++ {
		metadata := index.Entries[i]
		buf := make([]byte, 64*1024)
		req := NewReadRequestBuilder().
			WithOffset(metadata.FileOffset).
			WithLength(metadata.Length).
			WithBuffer(&buf).
			Build()

		reader.Read(context.Background(), req)
	}

	err = reader.Close()
	assert.NoError(t, err)

	reader.mu.Lock()
	assert.Nil(t, reader.pooledFiles)
	reader.mu.Unlock()
}

