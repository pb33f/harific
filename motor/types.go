package motor

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/pb33f/harhar"
)

type EntryMetadata struct {
	FileOffset   int64
	Length       int64
	Method       string
	URL          string
	StatusCode   int
	StatusText   string
	MimeType     string
	Timestamp    time.Time
	Duration     float64
	RequestSize  int64
	ResponseSize int64
	BodySize     int64
	PageRef      string
	ServerIP     string
	Connection   string
	HasError     bool
	IsCompressed bool
	IsCached     bool
}

type Index struct {
	FilePath           string
	FileSize           int64
	FileHash           string
	IndexVersion       int
	Version            string
	Creator            *harhar.Creator
	Browser            *harhar.Creator
	Pages              []harhar.Page
	Entries            []*EntryMetadata
	TotalEntries       int
	stringShards       [256]*stringTableShard
	TotalRequestBytes  int64
	TotalResponseBytes int64
	TimeRange          TimeRange
	UniqueURLs         int
	BuildTime          time.Duration
}

type stringTableShard struct {
	table map[string]string
	mu    sync.RWMutex
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// uses 256 shards with xxhash distribution to minimize lock contention during concurrent index building
func (idx *Index) Intern(s string) string {
	if s == "" {
		return ""
	}

	h := xxhash.Sum64String(s)
	shardIdx := h % 256
	shard := idx.stringShards[shardIdx]

	if shard == nil {
		idx.initShard(shardIdx)
		shard = idx.stringShards[shardIdx]
	}

	shard.mu.RLock()
	if interned, exists := shard.table[s]; exists {
		shard.mu.RUnlock()
		return interned
	}
	shard.mu.RUnlock()

	// double-checked locking: check without write lock first, then with write lock to prevent race
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if interned, exists := shard.table[s]; exists {
		return interned
	}

	shard.table[s] = s
	return s
}

func (idx *Index) initShard(shardIdx uint64) {
	if idx.stringShards[shardIdx] == nil {
		idx.stringShards[shardIdx] = &stringTableShard{
			table: make(map[string]string),
		}
	}
}

type StreamResult struct {
	Index int
	Entry *harhar.Entry
	Error error
}

type StreamerStats struct {
	TotalReads      int64
	CacheHits       int64
	CacheMisses     int64
	BytesRead       int64
	EntriesParsed   int64
	ParseErrors     int64
	AverageReadTime time.Duration
}

type StreamerOptions struct {
	ReadBufferSize int
	EnableCache    bool
	WorkerCount    int
}

func DefaultStreamerOptions() StreamerOptions {
	return StreamerOptions{
		ReadBufferSize: 64 * 1024,
		EnableCache:    false,
		WorkerCount:    4,
	}
}
