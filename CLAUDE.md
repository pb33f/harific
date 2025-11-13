# Braid - HAR File Analysis and Replay Tool

## Overview
Braid is a dual-purpose application for working with HAR (HTTP Archive) files:
1. **Terminal User Interface (TUI)** - Visualize and explore large HAR files
2. **Mock Server** - Replay captured HTTP responses to simulate original server behavior

## Architecture

### Motor Package (Core Engine)
The `motor` package is the high-performance streaming engine that powers Braid. It enables processing of massive HAR files (multiple gigabytes) without loading them entirely into memory.

#### Key Components:
- **Index Builder** (`index.go`) - Scans HAR files once to build a lightweight metadata index
- **Entry Reader** (`reader.go`) - Provides random access to individual HAR entries using file offsets
- **Streamer** (`streamer.go`) - High-level API for filtered/range-based streaming
- **Type System** (`types.go`) - Efficient data structures with string interning
- **Interfaces** (`interfaces.go`) - Clean abstractions for extensibility

#### Performance Characteristics:
- Handles 5GB+ HAR files with ~47MB memory usage
- ~240μs random access time per entry
- Concurrent streaming with worker pools
- 71% test coverage

#### How It Works:
1. **Indexing Phase** - Builds a catalog of all entries with file offsets (one-time scan)
2. **Access Phase** - Uses the index to jump directly to any entry without reading the entire file
3. **Streaming Phase** - Efficiently processes subsets of entries based on filters or ranges

### Application Modes

#### 1. TUI Mode (Terminal User Interface)
- Browse through HTTP requests and responses
- View headers, bodies, and metadata in a readable format
- Navigate large HAR files efficiently
- Search and filter capabilities
- Real-time rendering of request/response data

#### 2. Mock Server Mode
- Acts as an HTTP server that replays captured responses
- Matches incoming requests to HAR entries
- Returns the original captured response
- Useful for:
  - Testing client applications without the original server
  - Reproducing exact server behavior from captured sessions
  - Offline development against recorded API responses
  - Performance testing with consistent responses

## Use Cases

1. **Debugging** - Analyze HTTP traffic captured during debugging sessions
2. **Performance Analysis** - Review timing data and response sizes
3. **Testing** - Create reproducible test environments using captured traffic
4. **Development** - Work offline with recorded API responses
5. **Security Analysis** - Examine requests and responses for security issues
6. **Documentation** - Document actual API behavior from real traffic

## Technical Design Decisions

### Memory Efficiency
- Stream processing instead of full file loading
- String interning with sharded hash tables
- Selective field parsing (skip response bodies when not needed)
- Lazy loading of entry data

### JSON Processing
- Currently uses Go's standard library `encoding/json`
- Abstracted behind `HARDecoder` interface for future flexibility
- Token-level parsing for selective field extraction
- Considered Sonic library but incompatible with token-level API needs

### Concurrency
- Worker pools for parallel entry processing
- Atomic statistics tracking
- Thread-safe string interning with sharded locks
- Context-aware operations for cancellation

## HAR File Format
HAR (HTTP Archive) files are JSON-formatted logs containing:
- Complete HTTP request details (method, URL, headers, body)
- Full HTTP response data (status, headers, body content)
- Timing information for each request phase
- Browser/client metadata
- Often contain base64-encoded binary content (images, etc.)

These files can grow to multiple gigabytes when capturing extended browsing sessions or API interactions.

## Development Guidelines

### Adding New Features
1. Maintain the streaming architecture - avoid loading entire files
2. Use the Index for metadata queries before accessing entries
3. Implement filters at the metadata level when possible
4. Add appropriate test coverage
5. Consider memory impact of new features

### Performance Considerations
- The Index is loaded once and kept in memory - keep it lightweight
- Entry reading should be lazy - only load what's needed
- Use worker pools for concurrent operations
- Track performance metrics for monitoring

## Future Enhancements
- [ ] Caching layer for frequently accessed entries
- [ ] Additional filter predicates for streaming
- [ ] Response modification capabilities for mock server
- [ ] Export capabilities (filtered HAR subsets)