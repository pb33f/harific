package motor

import (
	"testing"
	"time"
)

func TestIndex_Intern(t *testing.T) {
	idx := &Index{}

	s1 := idx.Intern("hello")
	s2 := idx.Intern("hello")

	if s1 != s2 {
		t.Error("expected same string reference for identical strings")
	}

	if &s1 == &s2 {
		t.Error("expected different variable addresses but same string pointer")
	}

	empty := idx.Intern("")
	if empty != "" {
		t.Errorf("expected empty string, got %q", empty)
	}

	s3 := idx.Intern("world")
	if s1 == s3 {
		t.Error("expected different string references for different strings")
	}
}

func TestIndex_InternConcurrent(t *testing.T) {
	idx := &Index{}

	done := make(chan bool)
	iterations := 100

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				idx.Intern("concurrent-test")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDefaultStreamerOptions(t *testing.T) {
	opts := DefaultStreamerOptions()

	if opts.ReadBufferSize != 64*1024 {
		t.Errorf("expected buffer size 64KB, got %d", opts.ReadBufferSize)
	}

	if opts.EnableCache {
		t.Error("expected cache disabled by default")
	}

	if opts.WorkerCount != 4 {
		t.Errorf("expected 4 workers, got %d", opts.WorkerCount)
	}
}

func TestEntryMetadata(t *testing.T) {
	timestamp := time.Now()

	meta := &EntryMetadata{
		FileOffset:   1024,
		Length:       2048,
		Method:       "GET",
		URL:          "https://api.example.com/test",
		StatusCode:   200,
		StatusText:   "OK",
		MimeType:     "application/json",
		Timestamp:    timestamp,
		Duration:     123.45,
		RequestSize:  512,
		ResponseSize: 1536,
		BodySize:     1024,
	}

	if meta.FileOffset != 1024 {
		t.Errorf("expected offset 1024, got %d", meta.FileOffset)
	}

	if meta.Method != "GET" {
		t.Errorf("expected method GET, got %s", meta.Method)
	}

	if meta.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", meta.StatusCode)
	}

	if meta.Duration != 123.45 {
		t.Errorf("expected duration 123.45, got %f", meta.Duration)
	}
}

func TestTimeRange(t *testing.T) {
	start := time.Now()
	end := start.Add(1 * time.Hour)

	tr := TimeRange{
		Start: start,
		End:   end,
	}

	if !tr.Start.Equal(start) {
		t.Error("expected start times to match")
	}

	if !tr.End.Equal(end) {
		t.Error("expected end times to match")
	}

	duration := tr.End.Sub(tr.Start)
	if duration != 1*time.Hour {
		t.Errorf("expected duration 1h, got %v", duration)
	}
}

func TestStreamResult(t *testing.T) {
	result := StreamResult{
		Index: 42,
		Entry: nil,
		Error: nil,
	}

	if result.Index != 42 {
		t.Errorf("expected index 42, got %d", result.Index)
	}

	if result.Entry != nil {
		t.Error("expected nil entry")
	}

	if result.Error != nil {
		t.Error("expected nil error")
	}
}

func TestStreamerStats(t *testing.T) {
	stats := StreamerStats{
		TotalReads:      100,
		CacheHits:       75,
		CacheMisses:     25,
		BytesRead:       1024 * 1024,
		EntriesParsed:   100,
		ParseErrors:     0,
		AverageReadTime: 5 * time.Millisecond,
	}

	if stats.TotalReads != 100 {
		t.Errorf("expected 100 reads, got %d", stats.TotalReads)
	}

	if stats.CacheHits != 75 {
		t.Errorf("expected 75 hits, got %d", stats.CacheHits)
	}

	if stats.CacheMisses != 25 {
		t.Errorf("expected 25 misses, got %d", stats.CacheMisses)
	}

	hitRate := float64(stats.CacheHits) / float64(stats.TotalReads)
	expectedHitRate := 0.75
	if hitRate != expectedHitRate {
		t.Errorf("expected hit rate %.2f, got %.2f", expectedHitRate, hitRate)
	}
}
