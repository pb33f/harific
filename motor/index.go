package motor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/pb33f/harhar"
)

const (
	keyLog     = "log"
	keyVersion = "version"
	keyCreator = "creator"
	keyBrowser = "browser"
	keyPages   = "pages"
	keyEntries = "entries"
)

type DefaultIndexBuilder struct {
	index      *Index
	hash       *xxhash.Digest
	byteReader *byteCountingReader
}

func NewIndexBuilder(filePath string) *DefaultIndexBuilder {
	return &DefaultIndexBuilder{
		index: &Index{
			FilePath:     filePath,
			Entries:      make([]*EntryMetadata, 0),
			IndexVersion: 1,
		},
		hash: xxhash.New(),
	}
}

func (b *DefaultIndexBuilder) Build(reader io.Reader) (*Index, error) {
	startTime := time.Now()

	// read entire content to buffer for byte tracking
	var buf bytes.Buffer
	teeReader := io.TeeReader(reader, &buf)

	// track bytes and hash while reading
	b.byteReader = &byteCountingReader{
		reader: teeReader,
		hash:   b.hash,
	}

	// read entire content
	fullContent, err := io.ReadAll(b.byteReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read har file: %w", err)
	}

	if err := b.parseHAR(bytes.NewReader(fullContent)); err != nil {
		return nil, fmt.Errorf("failed to parse har file: %w", err)
	}

	b.index.FileHash = fmt.Sprintf("%x", b.hash.Sum64())
	b.index.FileSize = int64(len(fullContent))
	b.index.FileContent = fullContent
	b.index.BuildTime = time.Since(startTime)
	b.index.TotalEntries = len(b.index.Entries)

	urlSet := make(map[string]struct{})
	for _, entry := range b.index.Entries {
		urlSet[entry.URL] = struct{}{}
	}
	b.index.UniqueURLs = len(urlSet)

	return b.index, nil
}

func (b *DefaultIndexBuilder) parseHAR(reader io.Reader) error {
	decoder := newHARDecoder(reader)

	if _, err := decoder.Token(); err != nil {
		return err
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case keyLog:
			if err := b.parseLog(decoder); err != nil {
				return err
			}
		default:
			if err := helper.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *DefaultIndexBuilder) parseLog(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		return err
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case keyVersion:
			if err := decoder.Decode(&b.index.Version); err != nil {
				return err
			}
		case keyCreator:
			var creator harhar.Creator
			if err := decoder.Decode(&creator); err != nil {
				return err
			}
			b.index.Creator = &creator
		case keyBrowser:
			var browser harhar.Creator
			if err := decoder.Decode(&browser); err != nil {
				return err
			}
			b.index.Browser = &browser
		case keyPages:
			if err := decoder.Decode(&b.index.Pages); err != nil {
				return err
			}
		case keyEntries:
			if err := b.parseEntries(decoder); err != nil {
				return err
			}
		default:
			if err := helper.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *DefaultIndexBuilder) parseEntries(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('[') {
		return fmt.Errorf("expected array delimiter, got %v", token)
	}

	entryIndex := 0
	for decoder.More() {
		startOffset := decoder.InputOffset()

		// parse only metadata fields by reading raw json and selective parsing
		metadata, err := b.parseEntryMetadata(decoder, entryIndex, startOffset)
		if err != nil {
			return fmt.Errorf("failed to parse entry %d: %w", entryIndex, err)
		}

		endOffset := decoder.InputOffset()
		metadata.Length = endOffset - startOffset

		b.index.Entries = append(b.index.Entries, metadata)
		b.index.TotalRequestBytes += metadata.RequestSize
		b.index.TotalResponseBytes += metadata.ResponseSize

		if b.index.TimeRange.Start.IsZero() || metadata.Timestamp.Before(b.index.TimeRange.Start) {
			b.index.TimeRange.Start = metadata.Timestamp
		}
		if metadata.Timestamp.After(b.index.TimeRange.End) {
			b.index.TimeRange.End = metadata.Timestamp
		}

		entryIndex++
	}

	return nil
}

func (b *DefaultIndexBuilder) parseEntryMetadata(decoder *json.Decoder, index int, startOffset int64) (*EntryMetadata, error) {
	var entry harhar.Entry
	if err := decoder.Decode(&entry); err != nil {
		return nil, err
	}

	metadata := &EntryMetadata{
		FileOffset: startOffset,
		Method:     b.index.Intern(entry.Request.Method),
		URL:        b.index.Intern(entry.Request.URL),
	}

	if entry.Start != "" {
		t, err := time.Parse(time.RFC3339, entry.Start)
		if err == nil {
			metadata.Timestamp = t
		}
	}

	if entry.Time > 0 {
		metadata.Duration = entry.Time
	}

	metadata.StatusCode = entry.Response.StatusCode
	metadata.StatusText = b.index.Intern(entry.Response.StatusText)

	if entry.Response.Body.MIMEType != "" {
		metadata.MimeType = b.index.Intern(entry.Response.Body.MIMEType)
		metadata.BodySize = int64(entry.Response.Body.Size)
	}

	metadata.ResponseSize = int64(entry.Response.BodySize)
	metadata.RequestSize = int64(entry.Request.BodySize)

	if entry.PageRef != "" {
		metadata.PageRef = b.index.Intern(entry.PageRef)
	}
	if entry.ServerIP != "" {
		metadata.ServerIP = b.index.Intern(entry.ServerIP)
	}
	if entry.Connection != "" {
		metadata.Connection = b.index.Intern(entry.Connection)
	}

	return metadata, nil
}

func (b *DefaultIndexBuilder) AddEntry(offset int64, metadata *EntryMetadata) error {
	metadata.FileOffset = offset
	b.index.Entries = append(b.index.Entries, metadata)
	return nil
}

func (b *DefaultIndexBuilder) HARificate() error {
	return nil
}

func (b *DefaultIndexBuilder) GetIndex() *Index {
	return b.index
}

type byteCountingReader struct {
	reader io.Reader
	hash   *xxhash.Digest
}

func (r *byteCountingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 && r.hash != nil {
		r.hash.Write(p[:n])
	}
	return n, err
}
