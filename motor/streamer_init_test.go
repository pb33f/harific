package motor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUninitializedStreamer_GetEntry tests that GetEntry returns error before initialization
func TestUninitializedStreamer_GetEntry(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	// Try to get an entry without initializing
	entry, err := streamer.GetEntry(context.Background(), 0)

	// Should get initialization error, not a panic
	assert.Nil(t, entry)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not initialized")
	assert.Contains(t, err.Error(), "call Initialize() first")
}

// TestUninitializedStreamer_GetMetadata tests that GetMetadata returns error before initialization
func TestUninitializedStreamer_GetMetadata(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	// Try to get metadata without initializing
	metadata, err := streamer.GetMetadata(0)

	// Should get initialization error, not a panic
	assert.Nil(t, metadata)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestUninitializedStreamer_StreamRange tests that StreamRange returns error before initialization
func TestUninitializedStreamer_StreamRange(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	// Try to stream without initializing
	resultChan, err := streamer.StreamRange(context.Background(), 0, 10)

	// Should get initialization error, not a panic
	assert.Nil(t, resultChan)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestUninitializedStreamer_StreamFiltered tests that StreamFiltered returns error before initialization
func TestUninitializedStreamer_StreamFiltered(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	filter := func(meta *EntryMetadata) bool { return true }

	// Try to stream without initializing
	resultChan, err := streamer.StreamFiltered(context.Background(), filter)

	// Should get initialization error, not a panic
	assert.Nil(t, resultChan)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestInitializedStreamer_WorksNormally tests that initialized streamer works as expected
func TestInitializedStreamer_WorksNormally(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	// Initialize properly
	err = streamer.Initialize(context.Background())
	require.Nil(t, err)

	defer streamer.Close()

	// Now GetEntry should work
	entry, err := streamer.GetEntry(context.Background(), 0)
	assert.Nil(t, err)
	assert.NotNil(t, entry)

	// GetMetadata should work
	metadata, err := streamer.GetMetadata(0)
	assert.Nil(t, err)
	assert.NotNil(t, metadata)

	// StreamRange should work
	resultChan, err := streamer.StreamRange(context.Background(), 0, 5)
	assert.Nil(t, err)
	assert.NotNil(t, resultChan)

	// Consume the results
	count := 0
	for range resultChan {
		count++
	}
	assert.Equal(t, 5, count)
}
