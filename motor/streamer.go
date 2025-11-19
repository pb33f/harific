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

	// cache not implemented yet - EnableCache option reserved for future use

	return streamer, nil
}

func (s *DefaultHARStreamer) Initialize(ctx context.Context) error {
	return s.InitializeWithProgress(ctx, nil)
}

func (s *DefaultHARStreamer) InitializeWithProgress(ctx context.Context, progressChan chan<- IndexProgress) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	file, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// get file size for progress tracking
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	builder := NewIndexBuilder(s.filePath)
	index, err := builder.BuildWithProgress(file, fileSize, progressChan)
	if err != nil {
		return fmt.Errorf("failed to build index: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
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
			atomic.AddInt64(&s.stats.totalReads, 1)
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

	return s.streamRange(ctx, start, end), nil
}

func (s *DefaultHARStreamer) StreamFiltered(ctx context.Context, filter func(*EntryMetadata) bool) (<-chan StreamResult, error) {
	var matchingIndices []int
	for i, metadata := range s.index.Entries {
		if filter(metadata) {
			matchingIndices = append(matchingIndices, i)
		}
	}

	return s.streamIndices(ctx, matchingIndices), nil
}

func (s *DefaultHARStreamer) streamRange(ctx context.Context, start, end int) <-chan StreamResult {
	resultChan := make(chan StreamResult, s.options.WorkerCount)

	go func() {
		defer close(resultChan)

		workerCount := s.options.WorkerCount
		if workerCount < 1 {
			workerCount = 1
		}

		var wg sync.WaitGroup
		workChan := make(chan int, workerCount*2)

		for i := 0; i < workerCount; i++ {
			wg.Go(func() {
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
			})
		}

	ProducerLoop:
		for idx := start; idx < end; idx++ {
			select {
			case <-ctx.Done():
				break ProducerLoop
			case workChan <- idx:
			}
		}
		close(workChan)

		wg.Wait()
	}()

	return resultChan
}

func (s *DefaultHARStreamer) streamIndices(ctx context.Context, indices []int) <-chan StreamResult {
	resultChan := make(chan StreamResult, s.options.WorkerCount)

	go func() {
		defer close(resultChan)

		workerCount := s.options.WorkerCount
		if workerCount < 1 {
			workerCount = 1
		}

		var wg sync.WaitGroup
		workChan := make(chan int, workerCount*2)

		for i := 0; i < workerCount; i++ {
			wg.Go(func() {
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
			})
		}

	ProducerLoop:
		for _, idx := range indices {
			select {
			case <-ctx.Done():
				break ProducerLoop
			case workChan <- idx:
			}
		}
		close(workChan)

		wg.Wait()
	}()

	return resultChan
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
