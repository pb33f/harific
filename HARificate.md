# HARificate.md: Complete Design Specification for the HARific Engine

## Executive Summary

The HARific engine is a high-performance, memory-efficient streaming parser for HTTP Archive (HAR) files, designed to handle files from kilobytes to gigabytes without loading them entirely into memory. This document provides a complete implementation blueprint for the `motor` module of the Braid application.

## Table of Contents
1. [Project Overview](#project-overview)
2. [Core Design Principles](#core-design-principles)
3. [Architecture Overview](#architecture-overview)
4. [Interface Specifications](#interface-specifications)
5. [Data Structures](#data-structures)
6. [Implementation Strategy](#implementation-strategy)
7. [Detailed Component Design](#detailed-component-design)
8. [Performance Specifications](#performance-specifications)
9. [Testing Strategy](#testing-strategy)
10. [Migration Path to Sonic](#migration-path-to-sonic)

## Project Overview

### Purpose
HARific is the core engine within the `motor` module that powers Braid's ability to explore massive HAR files through a terminal user interface. It provides lightning-fast random access to any entry within a HAR file without loading the entire file into memory.

### Key Requirements
- **Memory Efficiency**: Let Go's garbage collector handle memory naturally
- **Speed**: Sub-millisecond entry access after initial indexing
- **Scalability**: Handle HAR files from 1KB to 100GB+
- **Interface-Driven**: All components behind interfaces for future optimization
- **Local-First**: Optimized for local file system access
- **Optional Caching**: Simple LRU cache that can be enabled/disabled

### Module Structure
```
braid/
├── motor/                    # The motor module
│   ├── interfaces.go        # All interface definitions
│   ├── streamer.go          # Main streamer implementation
│   ├── index.go             # Index builder and manager
│   ├── parser.go            # Token-based JSON parser
│   ├── reader.go            # Entry reader implementation
│   ├── cache.go             # LRU cache implementation
│   └── types.go             # Shared types and structs
└── main.go                  # Application entry point
```

## Core Design Principles

### 1. Interface Segregation
Every major component is defined by an interface, allowing implementations to be swapped without affecting consumers. This is critical for our planned migration from standard library JSON parsing to Sonic.

### 2. Two-Phase Processing
- **Phase 1**: Build a lightweight index by scanning the file once
- **Phase 2**: Use the index for O(1) random access to any entry

### 3. Lazy Loading
Never load data unless explicitly requested. Response bodies, which constitute 95% of file size, are only loaded on demand.

### 4. Optional Caching
Simple LRU cache that can be enabled via configuration. When disabled, the system bypasses cache logic entirely.

### 5. Progressive Enhancement
Start with a working implementation using standard library, optimize based on profiling, upgrade to Sonic when needed.

## Architecture Overview

### System Layers

```
┌──────────────────────────────────────────────┐
│              Application Layer               │
│         (Braid TUI Application)              │
└─────────────────▲────────────────────────────┘
                  │ Uses
┌─────────────────▼────────────────────────────┐
│              Motor Module API                │
│          (High-Level Interface)              │
├───────────────────────────────────────────────┤
│              HARific Engine                  │
├───────────────────────────────────────────────┤
│         Orchestration Component              │
│   (Coordinates all subsystems below)         │
├───────────────────────────────────────────────┤
│  Index Manager │ Entry Reader │ Cache (opt)  │
├───────────────────────────────────────────────┤
│           Token Parser Component             │
├───────────────────────────────────────────────┤
│            File I/O Component                │
│         (Buffered Reader + Seeker)           │
└───────────────────────────────────────────────┘
```

### Data Flow

```
Initial Index Building:
HAR File → Buffered Reader → Token Parser → Index Builder → Index

Entry Access (with cache enabled):
Request → Cache Check → Hit: Return Cached
                      ↓ Miss
              Index Lookup → File Seek → Token Parser → Entry Object → Cache Store → Response

Entry Access (cache disabled):
Request → Index Lookup → File Seek → Token Parser → Entry Object → Response
```

## Interface Specifications

### Primary Interfaces

#### 1. HARStreamer (Main Interface)
```go
package motor

import (
    "context"
    "github.com/pb33f/harhar"
)

// HARStreamer is the main interface for streaming HAR file entries
type HARStreamer interface {
    // Initialize builds the index from the HAR file
    // This is a one-time operation that must be called before any other methods
    Initialize(ctx context.Context) error

    // GetEntry retrieves a single entry by index (0-based)
    // Returns error if index is out of bounds
    GetEntry(ctx context.Context, index int) (*harhar.Entry, error)

    // StreamRange streams entries from start to end (exclusive)
    // Channel is closed when streaming completes or error occurs
    StreamRange(ctx context.Context, start, end int) (<-chan StreamResult, error)

    // StreamFiltered streams entries that match the filter function
    // Only entries whose metadata matches the filter are loaded and sent
    StreamFiltered(ctx context.Context, filter func(*EntryMetadata) bool) (<-chan StreamResult, error)

    // GetMetadata returns just the metadata without loading the full entry
    // This is extremely fast as metadata is stored in the index
    GetMetadata(index int) (*EntryMetadata, error)

    // GetIndex returns the full index for advanced queries
    GetIndex() *Index

    // Close releases all resources (file handles, memory, etc.)
    Close() error

    // Stats returns performance statistics
    Stats() StreamerStats
}
```

#### 2. IndexBuilder
```go
// IndexBuilder constructs the entry index during the initial scan
type IndexBuilder interface {
    // Build creates the index from a reader positioned at start of HAR file
    Build(reader io.Reader) (*Index, error)

    // AddEntry adds a single entry to the index during building
    // Called for each entry found during scanning
    AddEntry(offset int64, metadata *EntryMetadata) error

    // HARificate performs post-processing optimizations
    // Builds secondary indices, sorts data, interns strings
    HARificate() error

    // Marshal serializes the index for persistence
    Marshal() ([]byte, error)

    // Unmarshal deserializes a previously saved index
    Unmarshal(data []byte) error
}
```

#### 3. TokenParser
```go
// TokenParser provides low-level JSON token parsing
type TokenParser interface {
    // Init initializes the parser at a specific file position
    Init(reader io.ReadSeeker, offset int64) error

    // NavigateTo moves the parser to a specific JSON path
    // Example: ["log", "entries", "5"] navigates to the 5th entry
    NavigateTo(path []string) error

    // NextToken returns the next JSON token
    NextToken() (json.Token, error)

    // Skip skips the current JSON structure (object or array)
    Skip() error

    // Decode decodes the current position into the provided interface
    Decode(v interface{}) error

    // Position returns current byte position in the stream
    Position() int64

    // Depth returns current nesting depth in JSON structure
    Depth() int
}
```

#### 4. EntryReader
```go
// EntryReader handles reading individual entries from the file
type EntryReader interface {
    // ReadAt reads a complete entry at the specified offset
    ReadAt(offset int64, length int64) (*harhar.Entry, error)

    // ReadPartial reads only specific fields of an entry
    // Useful for reading just headers without response body
    ReadPartial(offset int64, fields []string) (map[string]interface{}, error)

    // StreamResponseBody returns a reader for the response body
    // Allows streaming large response bodies without loading into memory
    StreamResponseBody(offset int64) (io.ReadCloser, error)

    // ReadMetadata extracts just the metadata fields
    ReadMetadata(offset int64) (*EntryMetadata, error)
}
```

#### 5. Cache (Simple, Optional)
```go
// Cache provides optional caching for parsed entries
type Cache interface {
    // Get retrieves an entry from cache
    Get(index int) (*harhar.Entry, bool)

    // Put stores an entry in cache
    Put(index int, entry *harhar.Entry)

    // Clear removes all cached entries
    Clear()
}
```

## Data Structures

### Core Types

#### EntryMetadata
```go
// EntryMetadata contains frequently accessed fields stored in the index
// This structure is optimized for memory efficiency and quick access
type EntryMetadata struct {
    // File location
    FileOffset   int64  // Byte position where entry starts
    Length       int64  // Approximate length in bytes

    // HTTP request fields
    Method       string // GET, POST, etc.
    URL          string // Full URL

    // HTTP response fields
    StatusCode   int    // 200, 404, etc.
    StatusText   string // "OK", "Not Found", etc.
    MimeType     string // Content-Type value

    // Timing information
    Timestamp    time.Time // When request started
    Duration     float64   // Total time in milliseconds

    // Size information for memory planning
    RequestSize  int64 // Bytes in request (headers + body)
    ResponseSize int64 // Bytes in response (headers + body)
    BodySize     int64 // Just the response body size

    // References
    PageRef      string // Reference to page ID if grouped
    ServerIP     string // Server IP address
    Connection   string // Connection ID for reuse tracking

    // Optimization flags
    HasError     bool   // Quick error filtering
    IsCompressed bool   // Response was compressed
    IsCached     bool   // Response was from cache
}
```

#### Index
```go
// Index is the master structure for fast lookups
type Index struct {
    // File information
    FilePath     string
    FileSize     int64
    FileHash     string // Using xxHash64 for speed
    IndexVersion int    // For compatibility

    // HAR metadata from initial scan
    Version      string
    Creator      *harhar.Creator
    Browser      *harhar.Creator
    Pages        []harhar.Page // Lightweight page info

    // Primary entry index
    Entries      []*EntryMetadata
    TotalEntries int

    // Secondary indices for fast filtering
    URLIndex     map[string][]int      // URL → entry indices
    StatusIndex  map[int][]int         // Status code → entry indices
    MethodIndex  map[string][]int      // HTTP method → entry indices
    TimeIndex    *TimeRangeTree        // For time-based queries
    PageIndex    map[string][]int      // Page ID → entry indices

    // String interning table (memory optimization)
    StringTable  map[string]string     // Deduplicated strings

    // Statistics
    TotalRequestBytes  int64
    TotalResponseBytes int64
    TimeRange          TimeRange
    UniqueURLs         int

    // Internal
    buildTime    time.Duration
}
```

#### StreamResult
```go
// StreamResult is returned when streaming entries
type StreamResult struct {
    Index    int              // Entry index
    Entry    *harhar.Entry    // Full entry (nil if error)
    Metadata *EntryMetadata   // Always included
    Error    error            // Any error encountered
}
```

#### StreamerOptions
```go
// StreamerOptions configures the HARific engine
type StreamerOptions struct {
    // File handling
    ReadBufferSize  int    // Buffer for reading (default: 64KB)

    // Index options
    IndexCachePath  string // Where to save/load index
    RebuildIndex    bool   // Force rebuild even if cached

    // Cache configuration
    EnableCache     bool   // Enable LRU caching

    // Performance tuning
    WorkerCount     int    // Parallel workers for indexing
}
```

#### StreamerStats
```go
// StreamerStats provides performance metrics
type StreamerStats struct {
    // File operations
    TotalSeeks      int64
    BytesRead       int64
    ParseTime       time.Duration
    IndexBuildTime  time.Duration

    // Cache metrics (if enabled)
    CacheHits       int64
    CacheMisses     int64

    // Performance
    AverageSeekTime time.Duration
    AverageLoadTime time.Duration

    // Parser type in use
    ParserType      string // "stdlib" or "sonic"
}
```

#### Supporting Types
```go
// TimeRange represents a time span
type TimeRange struct {
    Start time.Time
    End   time.Time
}

// TimeRangeTree enables efficient time-based queries
type TimeRangeTree struct {
    // Implementation details for balanced tree
    // Enables O(log n) time range searches
}

// CacheStats provides cache performance metrics
type CacheStats struct {
    Hits    int64
    Misses  int64
    Size    int
}
```

## Implementation Strategy

### Phase 1: Core Implementation (Week 1)

#### Step 1: Define Interfaces
Create `interfaces.go` with all interface definitions

#### Step 2: Basic File I/O Layer
Create buffered file reader with configurable buffer size and seeking capability

#### Step 3: Token Parser with json.Decoder
Implement TokenParser interface using encoding/json

#### Step 4: Index Builder
- Scan HAR file and extract entry positions
- Parse minimal metadata for each entry
- Implement HARificate() for optimization
- Use xxHash for file validation

#### Step 5: Entry Reader
Implement EntryReader using TokenParser for selective loading

#### Step 6: Basic HARStreamer
Coordinate IndexBuilder and EntryReader with optional cache support

### Phase 2: Optimization (Week 2)

#### Step 7: Simple LRU Cache
Implement the Cache interface with basic LRU eviction

#### Step 8: Parallel Index Building
Use worker pool for faster metadata extraction

#### Step 9: Index Persistence
Save/load index with xxHash validation

### Phase 3: Advanced Features (Week 3)

#### Step 10: Advanced Querying
Time range queries and complex filtering

#### Step 11: Performance Monitoring
Detailed metrics and statistics

### Phase 4: Sonic Integration (Week 4)

#### Step 12: Sonic Parser Implementation
Create SonicTokenParser implementing TokenParser interface

#### Step 13: Runtime Strategy Selection
Auto-detect best parser based on file size and CPU capabilities

## Detailed Component Design

### IndexBuilder Implementation Flow

```
1. Open HAR file with buffered reader (64KB buffer)
2. Create xxHash digest for file validation
3. Parse JSON structure:
   {"log": {
     "version": "...",
     "creator": {...},
     "pages": [...],
     "entries": [  ← Target
       {...},      ← Extract metadata + offset
       {...},      ← Extract metadata + offset
       ...
     ]
   }}
4. For each entry:
   - Record file offset
   - Extract key metadata (URL, method, status, size, time)
   - Skip response body content
   - Add to index
5. Call HARificate():
   - Build secondary indices (URL, status, method)
   - Create time range tree
   - Intern duplicate strings
   - Calculate statistics
6. Return complete Index
```

### Cache Integration Pattern

```go
type harStreamer struct {
    index   *Index
    reader  EntryReader
    cache   Cache  // nil if caching disabled
    file    *os.File
    options StreamerOptions
}

func (h *harStreamer) GetEntry(ctx context.Context, index int) (*harhar.Entry, error) {
    // Fast path: check cache if enabled
    if h.cache != nil {
        if entry, found := h.cache.Get(index); found {
            h.stats.CacheHits++
            return entry, nil
        }
        h.stats.CacheMisses++
    }

    // Get metadata from index (always available)
    metadata := h.index.Entries[index]

    // Load from disk
    entry, err := h.reader.ReadAt(metadata.FileOffset, metadata.Length)
    if err != nil {
        return nil, err
    }

    // Store in cache if enabled
    if h.cache != nil {
        h.cache.Put(index, entry)
    }

    return entry, nil
}
```

### Token Parser State Machine

```
States:
ROOT → LOG → ENTRIES → ENTRY[n] → {REQUEST|RESPONSE|TIMINGS|...}

Navigation is stack-based:
- Push state when entering object/array
- Pop state when exiting
- Track depth for efficient skipping
```

## Performance Specifications

### xxHash Performance Advantage

| Algorithm | Speed | Use Case |
|-----------|-------|----------|
| xxHash64 | 13 GB/s | File validation, index caching |
| MD5 | 450 MB/s | Not recommended |
| SHA-256 | 200 MB/s | Security, not needed here |

### Target Performance Metrics

| Operation | File Size | Target Time | Notes |
|-----------|-----------|-------------|-------|
| Index Build | 100MB | < 1 second | Using xxHash |
| Index Build | 1GB | < 10 seconds | Parallel possible |
| Index Build | 10GB | < 100 seconds | Stream processing |
| Get Entry (cached) | Any | < 0.01ms | Memory access |
| Get Entry (uncached) | Any | < 2ms | SSD seek + parse |
| Stream 100 entries | Any | < 200ms | Sequential optimization |

### Memory Usage

Without explicit memory management, let Go handle:
- Index: ~200 bytes per entry (efficient)
- Cache: Grows as needed (when enabled)
- Buffers: Reused automatically by GC

## Testing Strategy

### Unit Tests

1. **Parser Tests**
   - Token navigation
   - Skip functionality
   - Decode accuracy

2. **Index Tests**
   - Metadata extraction
   - Offset calculation
   - HARificate optimization
   - xxHash validation

3. **Cache Tests**
   - LRU eviction
   - Hit/miss tracking
   - Concurrent access

4. **Integration Tests**
   - End-to-end streaming
   - Various file sizes
   - Cache enabled/disabled
   - Performance benchmarks

### Test Files
- tiny.har (10 entries)
- small.har (100 entries)
- medium.har (1,000 entries)
- large.har (10,000 entries)
- huge.har (100,000 entries)

## Migration Path to Sonic

### Implementation Strategy

1. **Create SonicTokenParser**
   ```go
   type SonicTokenParser struct {
       // Implements TokenParser interface
       // Uses sonic.Unmarshal internally
   }
   ```

2. **Factory Pattern**
   ```go
   func NewTokenParser(parserType string) TokenParser {
       switch parserType {
       case "sonic":
           return &SonicTokenParser{}
       default:
           return &StdlibTokenParser{}
       }
   }
   ```

3. **Runtime Selection**
   - Auto-detect CPU SIMD support
   - Choose based on file size
   - Allow manual override

### No Breaking Changes
Since everything is interface-based, Sonic integration requires:
- No changes to HARStreamer
- No changes to IndexBuilder
- No changes to EntryReader
- Just swap the TokenParser implementation

## Implementation Checklist

### Phase 1: Core
- [ ] Create motor module structure
- [ ] Define all interfaces in interfaces.go
- [ ] Implement StdlibTokenParser
- [ ] Implement IndexBuilder with xxHash
- [ ] Implement EntryReader
- [ ] Implement basic HARStreamer (cache = nil)
- [ ] Add types.go with all data structures
- [ ] Write unit tests

### Phase 2: Optimization
- [ ] Implement LRU Cache
- [ ] Wire cache into HARStreamer
- [ ] Add parallel index building
- [ ] Implement index persistence

### Phase 3: Enhancement
- [ ] Add time range queries
- [ ] Add complex filtering
- [ ] Add performance monitoring
- [ ] Complete integration tests

### Phase 4: Sonic
- [ ] Implement SonicTokenParser
- [ ] Add parser factory
- [ ] Add runtime selection
- [ ] Benchmark both implementations

## Success Criteria

1. **Correctness**
   - Accurately parses all valid HAR files
   - Handles malformed files gracefully

2. **Performance**
   - Sub-2ms random access
   - Handles 10GB+ files
   - xxHash validation < 100ms for 1GB

3. **Simplicity**
   - Clean interface boundaries
   - Optional features (cache)
   - No unnecessary complexity

4. **Extensibility**
   - Easy Sonic integration
   - Room for future optimizations
   - Clear upgrade path

## Conclusion

The HARific engine provides a clean, efficient solution for streaming massive HAR files. Key design decisions:

- **Interface-driven** for future flexibility
- **Two-phase processing** for O(1) access
- **xxHash** for 30x faster validation
- **Optional caching** keeps it simple
- **No memory management** lets Go handle it

This design balances performance with simplicity, providing a solid foundation that can be enhanced incrementally without breaking changes.

---

**Ready to build!** This document provides everything needed to implement the motor module with the HARific engine.