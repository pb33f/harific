package motor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeWithProgress_ChannelClosedOnError tests that progress channel is closed on error
func TestInitializeWithProgress_ChannelClosedOnError(t *testing.T) {
	// Test various error scenarios

	t.Run("file not found", func(t *testing.T) {
		streamer, err := NewHARStreamer("/nonexistent/file.har", DefaultStreamerOptions())
		require.Nil(t, err)

		progressChan := make(chan IndexProgress, 10)
		channelClosed := make(chan bool, 1)

		// Monitor channel closure in goroutine
		go func() {
			for range progressChan {
				// Consume any progress updates
			}
			channelClosed <- true
		}()

		// This should fail and close the channel
		err = streamer.InitializeWithProgress(context.Background(), progressChan)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to open file")

		// Wait for channel to be closed (with timeout to prevent hanging test)
		select {
		case <-channelClosed:
			// Good - channel was closed
		case <-time.After(1 * time.Second):
			t.Fatal("progress channel was not closed after error")
		}
	})

	t.Run("invalid har content", func(t *testing.T) {
		// Create a temporary file with invalid content
		tmpDir := t.TempDir()
		invalidFile := filepath.Join(tmpDir, "invalid.har")
		err := os.WriteFile(invalidFile, []byte("not valid json"), 0644)
		require.Nil(t, err)

		streamer, err := NewHARStreamer(invalidFile, DefaultStreamerOptions())
		require.Nil(t, err)

		progressChan := make(chan IndexProgress, 10)
		channelClosed := make(chan bool, 1)

		// Monitor channel closure
		go func() {
			for range progressChan {
				// Consume any progress updates
			}
			channelClosed <- true
		}()

		// This should fail and close the channel
		err = streamer.InitializeWithProgress(context.Background(), progressChan)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to build index")

		// Wait for channel to be closed
		select {
		case <-channelClosed:
			// Good - channel was closed
		case <-time.After(1 * time.Second):
			t.Fatal("progress channel was not closed after build error")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		harFile, cleanup, err := generateSmallHAR()
		require.Nil(t, err)
		defer cleanup()

		streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
		require.Nil(t, err)

		progressChan := make(chan IndexProgress, 10)
		channelClosed := make(chan bool, 1)

		// Monitor channel closure
		go func() {
			for range progressChan {
				// Consume any progress updates
			}
			channelClosed <- true
		}()

		// Cancel context immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// This should fail and close the channel
		err = streamer.InitializeWithProgress(ctx, progressChan)
		assert.NotNil(t, err)
		assert.Equal(t, context.Canceled, err)

		// Wait for channel to be closed
		select {
		case <-channelClosed:
			// Good - channel was closed
		case <-time.After(1 * time.Second):
			t.Fatal("progress channel was not closed after context cancellation")
		}
	})
}

// TestInitializeWithProgress_ChannelClosedOnSuccess tests that progress channel is closed on success
func TestInitializeWithProgress_ChannelClosedOnSuccess(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	progressChan := make(chan IndexProgress, 100)
	channelClosed := make(chan bool, 1)
	progressCount := 0

	// Monitor channel closure and count progress updates
	go func() {
		for range progressChan {
			progressCount++
		}
		channelClosed <- true
	}()

	// This should succeed and close the channel
	err = streamer.InitializeWithProgress(context.Background(), progressChan)
	assert.Nil(t, err)

	// Wait for channel to be closed
	select {
	case <-channelClosed:
		// Good - channel was closed
		assert.Greater(t, progressCount, 0, "should have received some progress updates")
	case <-time.After(2 * time.Second):
		t.Fatal("progress channel was not closed after successful initialization")
	}

	// Verify streamer is usable
	assert.NotNil(t, streamer.GetIndex())
	assert.Greater(t, streamer.GetIndex().TotalEntries, 0)
}

// TestInitializeWithProgress_NilChannel tests that nil channel doesn't panic
func TestInitializeWithProgress_NilChannel(t *testing.T) {
	harFile, cleanup, err := generateSmallHAR()
	require.Nil(t, err)
	defer cleanup()

	streamer, err := NewHARStreamer(harFile, DefaultStreamerOptions())
	require.Nil(t, err)

	// Should not panic with nil channel
	err = streamer.InitializeWithProgress(context.Background(), nil)
	assert.Nil(t, err)

	// Verify streamer is usable
	assert.NotNil(t, streamer.GetIndex())
}