package motor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamRange_NoGoroutineLeak tests that goroutines are cleaned up when context is cancelled
func TestStreamRange_NoGoroutineLeak(t *testing.T) {
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 8 // Use multiple workers to test concurrent cleanup

	streamer, err := NewHARStreamer("../testdata/test-50MB.har", opts)
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Start streaming with context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	resultChan, err := streamer.StreamRange(ctx, 0, 100)
	require.Nil(t, err)

	// Read a few results
	count := 0
	for result := range resultChan {
		if result.Error == nil {
			count++
			if count >= 5 {
				// Cancel context while workers are still processing
				cancel()
				// Stop consuming results to test blocked send scenario
				break
			}
		}
	}

	// Give time for goroutines to clean up
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check that goroutines were cleaned up
	currentGoroutines := runtime.NumGoroutine()

	// Allow small variance (±2) for runtime goroutines
	assert.LessOrEqual(t, currentGoroutines, baselineGoroutines+2,
		"goroutine leak detected: baseline=%d, current=%d", baselineGoroutines, currentGoroutines)
}

// TestStreamFiltered_NoGoroutineLeak tests that filtered stream doesn't leak goroutines
func TestStreamFiltered_NoGoroutineLeak(t *testing.T) {
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 8

	streamer, err := NewHARStreamer("../testdata/test-50MB.har", opts)
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Start filtered streaming with context
	ctx, cancel := context.WithCancel(context.Background())

	filter := func(meta *EntryMetadata) bool {
		return meta.Method == "GET"
	}

	resultChan, err := streamer.StreamFiltered(ctx, filter)
	require.Nil(t, err)

	// Read a few results then cancel
	count := 0
	for result := range resultChan {
		if result.Error == nil {
			count++
			if count >= 3 {
				cancel()
				break // Stop consuming
			}
		}
	}

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	currentGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, currentGoroutines, baselineGoroutines+2,
		"goroutine leak in StreamFiltered: baseline=%d, current=%d", baselineGoroutines, currentGoroutines)
}

// TestStreamRange_ConsumerStopsReading tests that workers clean up when consumer stops reading
func TestStreamRange_ConsumerStopsReading(t *testing.T) {
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 4

	streamer, err := NewHARStreamer("../testdata/test-50MB.har", opts)
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Start streaming
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultChan, err := streamer.StreamRange(ctx, 0, 50)
	require.Nil(t, err)

	// Read only one result then abandon the channel
	firstResult := <-resultChan
	assert.Nil(t, firstResult.Error)

	// Don't read any more results - simulating consumer abandoning the stream
	// Context will timeout after 2 seconds, which should clean up workers

	// Wait for context timeout
	<-ctx.Done()
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Verify no goroutine leak
	currentGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, currentGoroutines, baselineGoroutines+2,
		"goroutine leak when consumer stops: baseline=%d, current=%d", baselineGoroutines, currentGoroutines)
}

// TestStreamRange_RapidCancellation tests rapid context cancellation doesn't cause issues
func TestStreamRange_RapidCancellation(t *testing.T) {
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 8

	streamer, err := NewHARStreamer("../testdata/test-50MB.har", opts)
	require.Nil(t, err)

	err = streamer.Initialize(context.Background())
	require.Nil(t, err)

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Rapidly start and cancel multiple streams
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		resultChan, err := streamer.StreamRange(ctx, 0, 20)
		require.Nil(t, err)

		// Cancel immediately or after reading one
		if i%2 == 0 {
			cancel() // Cancel immediately
		} else {
			<-resultChan // Read one
			cancel()     // Then cancel
		}

		// Drain any remaining results
		go func() {
			for range resultChan {
			}
		}()

		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all cleanup
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Verify no accumulating goroutines
	currentGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, currentGoroutines, baselineGoroutines+3,
		"goroutine leak after rapid cancellations: baseline=%d, current=%d", baselineGoroutines, currentGoroutines)
}