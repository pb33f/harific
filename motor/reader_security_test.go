package motor

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRead_EntrySizeLimit tests that entries exceeding MaxEntrySize are rejected
func TestRead_EntrySizeLimit(t *testing.T) {
	// Generate test HAR file
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	// Build a minimal index first
	file, err := os.Open(harFile)
	require.Nil(t, err, "failed to open test file")
	defer file.Close()

	builder := NewIndexBuilder(harFile)
	index, err := builder.Build(file)
	require.Nil(t, err, "failed to build index")

	// Create a reader with a valid index
	reader, err := NewEntryReader(harFile, index)
	require.Nil(t, err, "failed to create reader")
	defer reader.Close()

	// Create a request that exceeds the max size
	oversizedLength := int64(MaxEntrySize + 1)
	req := NewReadRequestBuilder().
		WithOffset(0).
		WithLength(oversizedLength).
		Build()

	// Attempt to read
	resp := reader.Read(context.Background(), req)

	// Should get an error about size limit
	assert.NotNil(t, resp.GetError(), "expected error for oversized entry")
	assert.Contains(t, resp.GetError().Error(), "exceeds maximum allowed size")
	assert.Contains(t, resp.GetError().Error(), "104857601") // MaxEntrySize + 1
	assert.Contains(t, resp.GetError().Error(), "104857600") // MaxEntrySize
}

// TestRead_BufferAllocationLimit tests that buffer allocation respects MaxEntrySize
func TestRead_BufferAllocationLimit(t *testing.T) {
	// Generate test HAR file
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	// Build a minimal index first
	file, err := os.Open(harFile)
	require.Nil(t, err, "failed to open test file")
	defer file.Close()

	builder := NewIndexBuilder(harFile)
	index, err := builder.Build(file)
	require.Nil(t, err, "failed to build index")

	// Create a reader with a valid index
	reader, err := NewEntryReader(harFile, index)
	require.Nil(t, err, "failed to create reader")
	defer reader.Close()

	// Create a buffer that would need to grow beyond max size
	smallBuf := make([]byte, 10)
	oversizedLength := int64(MaxEntrySize + 1)

	req := NewReadRequestBuilder().
		WithOffset(0).
		WithLength(oversizedLength).
		WithBuffer(&smallBuf).
		Build()

	// Attempt to read
	resp := reader.Read(context.Background(), req)

	// Should get an error about size limit before buffer allocation
	assert.NotNil(t, resp.GetError(), "expected error for oversized buffer allocation")
	assert.Contains(t, resp.GetError().Error(), "exceeds maximum")

	// Buffer should not have been reallocated
	assert.Equal(t, 10, cap(smallBuf), "buffer should not have grown")
}

// TestRead_ValidSizeBelowLimit tests that entries below MaxEntrySize work correctly
func TestRead_ValidSizeBelowLimit(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	// Build a small index for testing
	file, err := os.Open(harFile)
	require.Nil(t, err, "failed to open test file")
	defer file.Close()

	builder := NewIndexBuilder(harFile)
	index, err := builder.Build(file)
	require.Nil(t, err, "failed to build index")
	require.Greater(t, len(index.Entries), 0, "expected at least one entry")

	reader, err := NewEntryReader(harFile, index)
	require.Nil(t, err, "failed to create reader")
	defer reader.Close()

	// Get first entry metadata
	metadata := index.Entries[0]

	// Verify the entry size is reasonable
	assert.Less(t, metadata.Length, int64(MaxEntrySize), "test entry should be below limit")

	// Read the entry with a buffer
	buf := make([]byte, 0)
	req := NewReadRequestBuilder().
		WithOffset(metadata.FileOffset).
		WithLength(metadata.Length).
		WithBuffer(&buf).
		Build()

	resp := reader.Read(context.Background(), req)

	// Should succeed
	assert.Nil(t, resp.GetError(), "expected successful read for valid size")
	assert.NotNil(t, resp.GetEntry(), "expected entry to be read")
	assert.Greater(t, resp.GetBytesRead(), int64(0), "expected bytes to be read")
}