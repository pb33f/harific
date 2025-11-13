package motor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pb33f/harhar"
)

type DefaultEntryReader struct {
	filePath string
	file     *os.File
	index    *Index
	mu       sync.Mutex
}

func NewEntryReader(filePath string, index *Index) (*DefaultEntryReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &DefaultEntryReader{
		filePath: filePath,
		file:     file,
		index:    index,
	}, nil
}

func (r *DefaultEntryReader) ReadAt(offset int64, length int64) (*harhar.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.file.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	var limitedReader io.Reader
	if length > 0 {
		limitedReader = io.LimitReader(r.file, length)
	} else {
		limitedReader = r.file
	}

	// json array entries are comma-separated; when seeking to an offset mid-array, skip the separator to get valid json
	skipReader := &skipLeadingReader{reader: limitedReader}
	decoder := newHARDecoder(skipReader)

	var entry harhar.Entry
	if err := decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode entry: %w", err)
	}

	return &entry, nil
}

type skipLeadingReader struct {
	reader    io.Reader
	skipped   bool
	remainder []byte
}

func (s *skipLeadingReader) Read(p []byte) (n int, err error) {
	if !s.skipped {
		s.skipped = true
		buf := make([]byte, 64)
		n, err := s.reader.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		startIdx := 0
		for startIdx < n && (buf[startIdx] == ',' || buf[startIdx] == ' ' || buf[startIdx] == '\n' || buf[startIdx] == '\r' || buf[startIdx] == '\t') {
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

func (r *DefaultEntryReader) ReadPartial(offset int64, fields []string) (map[string]interface{}, error) {
	length := r.findLengthForOffset(offset)
	entry, err := r.ReadAt(offset, length)
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

func (r *DefaultEntryReader) StreamResponseBody(offset int64) (io.ReadCloser, error) {
	length := r.findLengthForOffset(offset)
	entry, err := r.ReadAt(offset, length)
	if err != nil {
		return nil, err
	}

	if entry.Response.Body.Content == "" {
		return io.NopCloser(strings.NewReader("")), nil
	}

	return io.NopCloser(strings.NewReader(entry.Response.Body.Content)), nil
}

func (r *DefaultEntryReader) findLengthForOffset(offset int64) int64 {
	for _, meta := range r.index.Entries {
		if meta.FileOffset == offset {
			return meta.Length
		}
	}
	return 0
}

func (r *DefaultEntryReader) ReadMetadata(offset int64) (*EntryMetadata, error) {
	for _, meta := range r.index.Entries {
		if meta.FileOffset == offset {
			return meta, nil
		}
	}
	return nil, fmt.Errorf("no entry found at offset %d", offset)
}

func (r *DefaultEntryReader) Close() error {
	if r.file != nil {
		err := r.file.Close()
		r.file = nil
		return err
	}
	return nil
}
