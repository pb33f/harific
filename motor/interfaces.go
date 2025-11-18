package motor

import (
    "context"
    "encoding/json"
    "io"

    "github.com/pb33f/harhar"
)

// HARStreamer is the main interface for streaming HAR file entries
// it provides random access and filtered streaming without loading
// the entire file into memory
type HARStreamer interface {
    // Initialize builds the index and prepares the streamer for reading
    Initialize(ctx context.Context) error

    // GetEntry retrieves a single entry by index with full metadata and body
    GetEntry(ctx context.Context, index int) (*harhar.Entry, error)

    // StreamRange streams entries within a specific index range [start, end)
    StreamRange(ctx context.Context, start, end int) (<-chan StreamResult, error)

    // StreamFiltered streams entries matching the provided filter function
    StreamFiltered(ctx context.Context, filter func(*EntryMetadata) bool) (<-chan StreamResult, error)

    // GetMetadata returns lightweight metadata for an entry without parsing the full entry
    GetMetadata(index int) (*EntryMetadata, error)

    // GetIndex returns the complete index for advanced querying
    GetIndex() *Index

    // Close releases all resources
    Close() error

    // Stats returns current streamer statistics
    Stats() StreamerStats
}

// HARDecoder provides JSON decoding abstraction for swapping between stdlib and sonic
type HARDecoder interface {
	// Token returns the next JSON token in the input stream
	Token() (json.Token, error)

	// Decode decodes the next JSON value into v
	Decode(v interface{}) error

	// More reports whether there is another element in the current array or object
	More() bool

	// InputOffset returns the input stream byte offset of the current decoder position
	InputOffset() int64
}

// IndexBuilder builds the lightweight index of all entries in a HAR file
type IndexBuilder interface {
    // Build constructs the index by scanning the entire HAR file
    Build(reader io.Reader) (*Index, error)

    // AddEntry adds an entry to the index (used during scanning)
    AddEntry(offset int64, metadata *EntryMetadata) error

    // GetIndex returns the completed index
    GetIndex() *Index
}

// EntryReader reads individual entries from specific file offsets
type EntryReader interface {
	// Read reads an entry using a request message
	Read(ctx context.Context, req ReadRequest) ReadResponse

	// ReadMetadata reads only the metadata without parsing the full entry
	ReadMetadata(offset int64) (*EntryMetadata, error)

	// Close releases all resources including file handles
	Close() error
}

// ReadRequest provides read-only access to entry read parameters
type ReadRequest interface {
	GetOffset() int64
	GetLength() int64
	GetBuffer() *[]byte // may be nil (fallback to direct read)
}

// ReadRequestBuilder constructs ReadRequest instances with fluent api
type ReadRequestBuilder interface {
	WithOffset(offset int64) ReadRequestBuilder
	WithLength(length int64) ReadRequestBuilder
	WithBuffer(buf *[]byte) ReadRequestBuilder
	Build() ReadRequest
}

// ReadResponse provides read-only access to read results
type ReadResponse interface {
	GetEntry() *harhar.Entry
	GetBytesRead() int64
	GetError() error
}

// Cache provides optional caching of parsed entries
// implementations should use LRU eviction strategy
type Cache interface {
    // Get retrieves an entry from cache
    Get(index int) (*harhar.Entry, bool)

    // Put stores an entry in cache
    Put(index int, entry *harhar.Entry)

    // Clear removes all entries from cache
    Clear()

    // Size returns the current number of cached entries
    Size() int
}

// Searcher provides efficient search capabilities across har entries
type Searcher interface {
	// Search executes a search and returns results via channel
	Search(ctx context.Context, pattern string, opts SearchOptions) (<-chan []SearchResult, error)

	// Stats returns current search statistics
	Stats() SearchStats
}
