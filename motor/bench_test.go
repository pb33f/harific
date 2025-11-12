package motor

import (
	"context"
	"math/rand"
	"testing"
)

// benchmark random access pattern - simulates jumping to random entries
func BenchmarkRandomAccess_5MB(b *testing.B) {
	benchmarkRandomAccess(b, "../testdata/test-5MB.har")
}

func BenchmarkRandomAccess_50MB(b *testing.B) {
	benchmarkRandomAccess(b, "../testdata/test-50MB.har")
}

func BenchmarkRandomAccess_500MB(b *testing.B) {
	benchmarkRandomAccess(b, "../testdata/test-500MB.har")
}

func benchmarkRandomAccess(b *testing.B, filePath string) {
	// setup
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer(filePath, opts)
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		b.Fatalf("initialize failed: %v", err)
	}

	totalEntries := streamer.index.TotalEntries
	if totalEntries == 0 {
		b.Skip("no entries in file")
	}

	// pre-generate random indices
	indices := make([]int, b.N)
	for i := 0; i < b.N; i++ {
		indices[i] = rand.Intn(totalEntries)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// benchmark
	for i := 0; i < b.N; i++ {
		_, err := streamer.GetEntry(ctx, indices[i])
		if err != nil {
			b.Fatalf("get entry failed: %v", err)
		}
	}

	// report stats
	stats := streamer.Stats()
	b.ReportMetric(float64(stats.TotalReads), "reads")
	b.ReportMetric(float64(stats.BytesRead)/(1024*1024), "MB_read")
}

// benchmark sequential streaming pattern - simulates streaming chunks of entries
func BenchmarkSequentialStream_5MB(b *testing.B) {
	benchmarkSequentialStream(b, "../testdata/test-5MB.har", 10)
}

func BenchmarkSequentialStream_50MB(b *testing.B) {
	benchmarkSequentialStream(b, "../testdata/test-50MB.har", 50)
}

func BenchmarkSequentialStream_500MB(b *testing.B) {
	benchmarkSequentialStream(b, "../testdata/test-500MB.har", 100)
}

func benchmarkSequentialStream(b *testing.B, filePath string, chunkSize int) {
	// setup
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 4

	streamer, err := NewHARStreamer(filePath, opts)
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		b.Fatalf("initialize failed: %v", err)
	}

	totalEntries := streamer.index.TotalEntries
	if totalEntries == 0 {
		b.Skip("no entries in file")
	}

	b.ResetTimer()
	b.ReportAllocs()

	// benchmark
	for i := 0; i < b.N; i++ {
		start := (i * chunkSize) % totalEntries
		end := start + chunkSize
		if end > totalEntries {
			end = totalEntries
		}

		resultChan, err := streamer.StreamRange(ctx, start, end)
		if err != nil {
			b.Fatalf("stream range failed: %v", err)
		}

		// consume results
		count := 0
		for range resultChan {
			count++
		}
	}

	// report stats
	stats := streamer.Stats()
	b.ReportMetric(float64(stats.TotalReads), "reads")
	b.ReportMetric(float64(stats.EntriesParsed), "parsed")
	b.ReportMetric(float64(stats.AverageReadTime.Microseconds()), "avg_us")
}

// benchmark index building - measures how fast we can build the index
func BenchmarkIndexBuild_5MB(b *testing.B) {
	benchmarkIndexBuild(b, "../testdata/test-5MB.har")
}

func BenchmarkIndexBuild_50MB(b *testing.B) {
	benchmarkIndexBuild(b, "../testdata/test-50MB.har")
}

func BenchmarkIndexBuild_500MB(b *testing.B) {
	benchmarkIndexBuild(b, "../testdata/test-500MB.har")
}

func benchmarkIndexBuild(b *testing.B, filePath string) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		opts := DefaultStreamerOptions()
		streamer, err := NewHARStreamer(filePath, opts)
		if err != nil {
			b.Skipf("test file not found: %v", err)
		}

		ctx := context.Background()
		if err := streamer.Initialize(ctx); err != nil {
			b.Fatalf("initialize failed: %v", err)
		}

		if i == 0 {
			index := streamer.GetIndex()
			b.ReportMetric(float64(index.TotalEntries), "entries")
			b.ReportMetric(float64(index.FileSize)/(1024*1024), "MB")
		}

		streamer.Close()
	}
}

// benchmark metadata access - measures lightweight metadata retrieval
func BenchmarkMetadataAccess_5MB(b *testing.B) {
	benchmarkMetadataAccess(b, "../testdata/test-5MB.har")
}

func BenchmarkMetadataAccess_50MB(b *testing.B) {
	benchmarkMetadataAccess(b, "../testdata/test-50MB.har")
}

func BenchmarkMetadataAccess_500MB(b *testing.B) {
	benchmarkMetadataAccess(b, "../testdata/test-500MB.har")
}

func benchmarkMetadataAccess(b *testing.B, filePath string) {
	// setup
	opts := DefaultStreamerOptions()
	streamer, err := NewHARStreamer(filePath, opts)
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		b.Fatalf("initialize failed: %v", err)
	}

	totalEntries := streamer.index.TotalEntries
	if totalEntries == 0 {
		b.Skip("no entries in file")
	}

	// pre-generate random indices
	indices := make([]int, b.N)
	for i := 0; i < b.N; i++ {
		indices[i] = rand.Intn(totalEntries)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// benchmark
	for i := 0; i < b.N; i++ {
		_, err := streamer.GetMetadata(indices[i])
		if err != nil {
			b.Fatalf("get metadata failed: %v", err)
		}
	}
}

// benchmark filtered streaming - measures performance of filtering
func BenchmarkFilteredStream_5MB_ByMethod(b *testing.B) {
	benchmarkFilteredStream(b, "../testdata/test-5MB.har", func(meta *EntryMetadata) bool {
		return meta.Method == "GET"
	})
}

func BenchmarkFilteredStream_50MB_ByMethod(b *testing.B) {
	benchmarkFilteredStream(b, "../testdata/test-50MB.har", func(meta *EntryMetadata) bool {
		return meta.Method == "POST"
	})
}

func BenchmarkFilteredStream_5MB_ByStatus(b *testing.B) {
	benchmarkFilteredStream(b, "../testdata/test-5MB.har", func(meta *EntryMetadata) bool {
		return meta.StatusCode == 200
	})
}

func benchmarkFilteredStream(b *testing.B, filePath string, filter func(*EntryMetadata) bool) {
	// setup
	opts := DefaultStreamerOptions()
	opts.WorkerCount = 4

	streamer, err := NewHARStreamer(filePath, opts)
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		b.Fatalf("initialize failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// benchmark
	for i := 0; i < b.N; i++ {
		resultChan, err := streamer.StreamFiltered(ctx, filter)
		if err != nil {
			b.Fatalf("stream filtered failed: %v", err)
		}

		// consume results
		count := 0
		for range resultChan {
			count++
		}

		// report matched count on first iteration
		if i == 0 {
			b.ReportMetric(float64(count), "matched")
		}
	}
}

func BenchmarkStringInterning(b *testing.B) {
	index := &Index{}

	urls := []string{
		"https://api.example.com/users",
		"https://api.example.com/posts",
		"https://api.example.com/comments",
		"https://cdn.example.com/images/avatar.png",
		"https://cdn.example.com/js/app.js",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		_ = index.Intern(url)
	}
}

// benchmark concurrent access - measures thread-safety overhead
func BenchmarkConcurrentAccess_5MB(b *testing.B) {
	benchmarkConcurrentAccess(b, "../testdata/test-5MB.har", 4)
}

func BenchmarkConcurrentAccess_50MB(b *testing.B) {
	benchmarkConcurrentAccess(b, "../testdata/test-50MB.har", 8)
}

func benchmarkConcurrentAccess(b *testing.B, filePath string, workers int) {
	// setup
	opts := DefaultStreamerOptions()
	opts.WorkerCount = workers

	streamer, err := NewHARStreamer(filePath, opts)
	if err != nil {
		b.Skipf("test file not found: %v", err)
	}
	defer streamer.Close()

	ctx := context.Background()
	if err := streamer.Initialize(ctx); err != nil {
		b.Fatalf("initialize failed: %v", err)
	}

	totalEntries := streamer.index.TotalEntries
	if totalEntries == 0 {
		b.Skip("no entries in file")
	}

	b.ResetTimer()
	b.ReportAllocs()

	// benchmark with concurrent goroutines
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := rand.Intn(totalEntries)
			_, err := streamer.GetEntry(ctx, idx)
			if err != nil {
				b.Fatalf("get entry failed: %v", err)
			}
		}
	})
}
