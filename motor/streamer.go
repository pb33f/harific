package motor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pb33f/harhar"
)

type DefaultHARStreamer struct {
	filePath string
	options  StreamerOptions
	index    *Index
	reader   *DefaultEntryReader
	cache    Cache
	stats    atomicStats
}

type atomicStats struct {
	totalReads      int64
	cacheHits       int64
	cacheMisses     int64
	bytesRead       int64
	entriesParsed   int64
	parseErrors     int64
	totalReadTimeNs int64
}

func NewHARStreamer(filePath string, options StreamerOptions) (*DefaultHARStreamer, error) {
	streamer := &DefaultHARStreamer{
		filePath: filePath,
		options:  options,
	}

	if options.EnableCache {
		streamer.cache = NewNoOpCache()
	}

	return streamer, nil
}

func (s *DefaultHARStreamer) Initialize(ctx context.Context) error {
	file, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	builder := NewIndexBuilder(s.filePath)
	index, err := builder.Build(file)
	if err != nil {
		return fmt.Errorf("failed to build index: %w", err)
	}

	s.index = index

	reader, err := NewEntryReader(s.filePath, s.index)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	s.reader = reader

	return nil
}

func (s *DefaultHARStreamer) GetEntry(ctx context.Context, index int) (*harhar.Entry, error) {
	if index < 0 || index >= s.index.TotalEntries {
		return nil, fmt.Errorf("index %d out of range [0, %d)", index, s.index.TotalEntries)
	}

	start := time.Now()

	if s.cache != nil {
		if entry, ok := s.cache.Get(index); ok {
			atomic.AddInt64(&s.stats.cacheHits, 1)
			return entry, nil
		}
		atomic.AddInt64(&s.stats.cacheMisses, 1)
	}

	metadata := s.index.Entries[index]
	entry, err := s.reader.ReadAt(metadata.FileOffset, metadata.Length)
	if err != nil {
		atomic.AddInt64(&s.stats.parseErrors, 1)
		return nil, fmt.Errorf("failed to read entry: %w", err)
	}

	atomic.AddInt64(&s.stats.totalReads, 1)
	atomic.AddInt64(&s.stats.entriesParsed, 1)
	atomic.AddInt64(&s.stats.bytesRead, metadata.Length)

	elapsed := time.Since(start)
	atomic.AddInt64(&s.stats.totalReadTimeNs, int64(elapsed))

	if s.cache != nil {
		s.cache.Put(index, entry)
	}

	return entry, nil
}

func (s *DefaultHARStreamer) StreamRange(ctx context.Context, start, end int) (<-chan StreamResult, error) {
	if start < 0 || start >= s.index.TotalEntries {
		return nil, fmt.Errorf("start index %d out of range", start)
	}
	if end < start || end > s.index.TotalEntries {
		return nil, fmt.Errorf("end index %d out of range", end)
	}

	resultChan := make(chan StreamResult, s.options.WorkerCount)

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup
		workChan := make(chan int, s.options.WorkerCount*2)

		for i := 0; i < s.options.WorkerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range workChan {
					select {
					case <-ctx.Done():
						return
					default:
						entry, err := s.GetEntry(ctx, idx)
						resultChan <- StreamResult{
							Index: idx,
							Entry: entry,
							Error: err,
						}
					}
				}
			}()
		}

		for i := start; i < end; i++ {
			select {
			case <-ctx.Done():
				break
			case workChan <- i:
			}
		}
		close(workChan)

		wg.Wait()
	}()

	return resultChan, nil
}

func (s *DefaultHARStreamer) StreamFiltered(ctx context.Context, filter func(*EntryMetadata) bool) (<-chan StreamResult, error) {
	resultChan := make(chan StreamResult, s.options.WorkerCount)

	go func() {
		defer close(resultChan)

		var matchingIndices []int
		for i, metadata := range s.index.Entries {
			if filter(metadata) {
				matchingIndices = append(matchingIndices, i)
			}
		}

		var wg sync.WaitGroup
		workChan := make(chan int, s.options.WorkerCount*2)

		for i := 0; i < s.options.WorkerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range workChan {
					select {
					case <-ctx.Done():
						return
					default:
						entry, err := s.GetEntry(ctx, idx)
						resultChan <- StreamResult{
							Index: idx,
							Entry: entry,
							Error: err,
						}
					}
				}
			}()
		}

		for _, idx := range matchingIndices {
			select {
			case <-ctx.Done():
				break
			case workChan <- idx:
			}
		}
		close(workChan)

		wg.Wait()
	}()

	return resultChan, nil
}

func (s *DefaultHARStreamer) GetMetadata(index int) (*EntryMetadata, error) {
	if index < 0 || index >= s.index.TotalEntries {
		return nil, fmt.Errorf("index %d out of range", index)
	}

	return s.index.Entries[index], nil
}

func (s *DefaultHARStreamer) GetIndex() *Index {
	return s.index
}

func (s *DefaultHARStreamer) Close() error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}

func (s *DefaultHARStreamer) Stats() StreamerStats {
	totalReads := atomic.LoadInt64(&s.stats.totalReads)
	totalTimeNs := atomic.LoadInt64(&s.stats.totalReadTimeNs)

	var avgTime time.Duration
	if totalReads > 0 {
		avgTime = time.Duration(totalTimeNs / totalReads)
	}

	return StreamerStats{
		TotalReads:      totalReads,
		CacheHits:       atomic.LoadInt64(&s.stats.cacheHits),
		CacheMisses:     atomic.LoadInt64(&s.stats.cacheMisses),
		BytesRead:       atomic.LoadInt64(&s.stats.bytesRead),
		EntriesParsed:   atomic.LoadInt64(&s.stats.entriesParsed),
		ParseErrors:     atomic.LoadInt64(&s.stats.parseErrors),
		AverageReadTime: avgTime,
	}
}

func (s *DefaultHARStreamer) HARificate() error {
	return nil
}
