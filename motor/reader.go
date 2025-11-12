package motor

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pb33f/harhar"
)

type DefaultEntryReader struct {
	index *Index
}

func NewEntryReader(filePath string, index *Index) (*DefaultEntryReader, error) {
	return &DefaultEntryReader{
		index: index,
	}, nil
}

func (r *DefaultEntryReader) ReadAt(offset int64, length int64) (*harhar.Entry, error) {
	if offset < 0 || offset >= int64(len(r.index.FileContent)) {
		return nil, fmt.Errorf("offset %d out of range", offset)
	}

	var content []byte
	if length > 0 && offset+length <= int64(len(r.index.FileContent)) {
		content = r.index.FileContent[offset : offset+length]
	} else {
		content = r.index.FileContent[offset:]
	}

	// skip leading commas/whitespace that may be part of json array
	startIdx := 0
	for startIdx < len(content) && (content[startIdx] == ',' || content[startIdx] == ' ' || content[startIdx] == '\n' || content[startIdx] == '\r' || content[startIdx] == '\t') {
		startIdx++
	}

	if startIdx > 0 {
		content = content[startIdx:]
	}

	decoder := newHARDecoder(bytes.NewReader(content))

	var entry harhar.Entry
	if err := decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode entry: %w", err)
	}

	return &entry, nil
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
		return io.NopCloser(nil), nil
	}

	return io.NopCloser(nil), nil
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
	entryIndex := int(offset)
	if entryIndex < 0 || entryIndex >= len(r.index.Entries) {
		return nil, fmt.Errorf("entry index %d out of range", entryIndex)
	}

	return r.index.Entries[entryIndex], nil
}

func (r *DefaultEntryReader) Close() error {
	return nil
}
