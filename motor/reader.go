package motor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pb33f/harhar"
)

const (
	// MaxEntrySize is the maximum size of a single HAR entry that can be read.
	// This prevents OOM attacks from malicious or corrupted HAR files.
	// 100MB should be sufficient for even very large responses.
	MaxEntrySize = 100 * 1024 * 1024 // 100MB
)

type DefaultEntryReader struct {
	filePath    string
	filePool    *sync.Pool               // pool of file handles for concurrent access
	index       *Index
	offsetIndex map[int64]*EntryMetadata // o(1) metadata lookup by offset
	pooledFiles []*os.File               // track pooled files for cleanup
	mu          sync.Mutex               // protects pooledFiles slice
}

// pooledFile wraps *os.File with thread-safe registration
type pooledFile struct {
	file   *os.File
	reader *DefaultEntryReader
	once   sync.Once
}

func (pf *pooledFile) register() {
	pf.once.Do(func() {
		pf.reader.mu.Lock()
		pf.reader.pooledFiles = append(pf.reader.pooledFiles, pf.file)
		pf.reader.mu.Unlock()
	})
}

func (pf *pooledFile) Seek(offset int64, whence int) (int64, error) {
	return pf.file.Seek(offset, whence)
}

func (pf *pooledFile) Read(p []byte) (n int, err error) {
	return pf.file.Read(p)
}

func NewEntryReader(filePath string, index *Index) (*DefaultEntryReader, error) {
	// pre-build offset index for o(1) metadata lookups during search
	offsetIndex := make(map[int64]*EntryMetadata, len(index.Entries))
	for i := range index.Entries {
		offsetIndex[index.Entries[i].FileOffset] = index.Entries[i]
	}

	reader := &DefaultEntryReader{
		filePath:    filePath,
		index:       index,
		offsetIndex: offsetIndex,
		pooledFiles: make([]*os.File, 0, 16),
	}

	reader.filePool = &sync.Pool{
		New: func() interface{} {
			file, err := os.Open(filePath)
			if err != nil {
				return nil
			}

			pf := &pooledFile{
				file:   file,
				reader: reader,
			}
			pf.register() // thread-safe registration using sync.once
			return pf
		},
	}

	return reader, nil
}

func (r *DefaultEntryReader) Read(ctx context.Context, req ReadRequest) ReadResponse {
	resp := newReadResponse()

	// Check context before starting expensive operations
	select {
	case <-ctx.Done():
		resp.err = ctx.Err()
		return resp
	default:
	}

	// Validate entry size to prevent OOM attacks
	if req.GetLength() > MaxEntrySize {
		resp.err = fmt.Errorf("entry size %d exceeds maximum allowed size %d", req.GetLength(), MaxEntrySize)
		return resp
	}

	// each worker gets isolated file handle from pool (no mutex contention)
	pooledHandle := r.filePool.Get()
	if pooledHandle == nil {
		resp.err = fmt.Errorf("failed to get file handle from pool")
		return resp
	}

	pf, ok := pooledHandle.(*pooledFile)
	if !ok || pf == nil {
		resp.err = fmt.Errorf("invalid file handle type")
		return resp
	}
	defer r.filePool.Put(pf)

	_, err := pf.Seek(req.GetOffset(), io.SeekStart)
	if err != nil {
		resp.err = fmt.Errorf("seek failed: %w", err)
		return resp
	}

	limitReader := io.LimitReader(pf, req.GetLength())

	var jsonReader io.Reader

	if buf := req.GetBuffer(); buf != nil {
		// search path: use pooled buffer for maximum efficiency
		if req.GetLength() > int64(cap(*buf)) {
			// Double-check size limit before allocation (defense in depth)
			if req.GetLength() > MaxEntrySize {
				resp.err = fmt.Errorf("cannot allocate buffer: entry size %d exceeds maximum %d", req.GetLength(), MaxEntrySize)
				return resp
			}
			*buf = make([]byte, req.GetLength())
		}

		n, err := io.ReadFull(limitReader, (*buf)[:req.GetLength()])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			resp.err = fmt.Errorf("read failed: %w", err)
			return resp
		}
		resp.bytesRead = int64(n)
		jsonReader = bytes.NewReader((*buf)[:n])
	} else {
		// fallback path: tui/serve use cases (not search)
		// direct read without buffer - less efficient but backward compatible
		jsonReader = limitReader
		resp.bytesRead = req.GetLength()
	}

	// Check context again before expensive JSON decode
	select {
	case <-ctx.Done():
		resp.err = ctx.Err()
		return resp
	default:
	}

	skipReader := &skipLeadingReader{reader: jsonReader}
	decoder := json.NewDecoder(skipReader)
	var entry harhar.Entry
	if err := decoder.Decode(&entry); err != nil {
		resp.err = fmt.Errorf("decode failed: %w", err)
		return resp
	}

	resp.entry = &entry
	return resp
}

// fast metadata lookup without loading full entry from disk
func (r *DefaultEntryReader) ReadMetadata(offset int64) (*EntryMetadata, error) {
	if meta, ok := r.offsetIndex[offset]; ok {
		return meta, nil
	}
	return nil, fmt.Errorf("metadata not found for offset %d", offset)
}

// close releases all file handles in the pool
func (r *DefaultEntryReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for _, file := range r.pooledFiles {
		if err := file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	r.pooledFiles = nil
	return firstErr
}

// skipLeadingReader skips json array separators (commas, whitespace)
type skipLeadingReader struct {
	reader    io.Reader
	skipped   bool
	remainder []byte
}

func (s *skipLeadingReader) Read(p []byte) (n int, err error) {
	if !s.skipped {
		s.skipped = true
		var buf [64]byte // stack allocated
		n, err := s.reader.Read(buf[:])
		if err != nil && err != io.EOF {
			return 0, err
		}

		startIdx := 0
		for startIdx < n && (buf[startIdx] == ',' || buf[startIdx] == ' ' ||
			buf[startIdx] == '\n' || buf[startIdx] == '\r' || buf[startIdx] == '\t') {
			startIdx++
		}

		cleaned := buf[startIdx:n]
		copied := copy(p, cleaned)
		if copied < len(cleaned) {
			s.remainder = make([]byte, len(cleaned)-copied)
			copy(s.remainder, cleaned[copied:])
		}
		return copied, nil
	}

	if len(s.remainder) > 0 {
		n = copy(p, s.remainder)
		s.remainder = s.remainder[n:]
		return n, nil
	}

	return s.reader.Read(p)
}

// deprecated: use Read() instead
func (r *DefaultEntryReader) ReadAt(offset int64, length int64) (*harhar.Entry, error) {
	req := NewReadRequestBuilder().
		WithOffset(offset).
		WithLength(length).
		Build()

	resp := r.Read(context.Background(), req)
	if resp.GetError() != nil {
		return nil, resp.GetError()
	}
	return resp.GetEntry(), nil
}

// deprecated: use Read() with entry.Response.Body.Content
func (r *DefaultEntryReader) StreamResponseBody(offset int64) (io.ReadCloser, error) {
	meta, err := r.ReadMetadata(offset)
	if err != nil {
		return nil, err
	}

	entry, err := r.ReadAt(offset, meta.Length)
	if err != nil {
		return nil, err
	}

	if entry.Response.Body.Content == "" {
		return io.NopCloser(strings.NewReader("")), nil
	}

	return io.NopCloser(strings.NewReader(entry.Response.Body.Content)), nil
}

// deprecated: use Read() and extract fields manually
func (r *DefaultEntryReader) ReadPartial(offset int64, fields []string) (map[string]interface{}, error) {
	meta, err := r.ReadMetadata(offset)
	if err != nil {
		return nil, err
	}

	entry, err := r.ReadAt(offset, meta.Length)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, field := range fields {
		switch field {
		case "request":
			result["request"] = entry.Request
		case "response":
			result["response"] = entry.Response
		case "timings":
			result["timings"] = entry.Timings
		case "startedDateTime":
			result["startedDateTime"] = entry.Start
		case "time":
			result["time"] = entry.Time
		}
	}

	return result, nil
}
