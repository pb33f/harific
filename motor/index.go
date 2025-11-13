package motor

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/pb33f/harhar"
)

const (
	keyLog               = "log"
	keyVersion           = "version"
	keyCreator           = "creator"
	keyBrowser           = "browser"
	keyPages             = "pages"
	keyEntries           = "entries"
	keyStartedDateTime   = "startedDateTime"
	keyTime              = "time"
	keyRequest           = "request"
	keyResponse          = "response"
	keyPageRef           = "pageref"
	keyServerIPAddress   = "serverIPAddress"
	keyConnection        = "connection"
	keyMethod            = "method"
	keyURL               = "url"
	keyBodySize          = "bodySize"
	keyStatus            = "status"
	keyStatusText        = "statusText"
	keyContent           = "content"
	keySize              = "size"
	keyMimeType          = "mimeType"
	keyText              = "text"
	keyEncoding          = "encoding"
)

type DefaultIndexBuilder struct {
	index     *Index
	hash      *xxhash.Digest
	bytesRead int64
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

	hashReader := &hashingReader{
		reader: reader,
		hash:   b.hash,
	}

	if err := b.parseHAR(hashReader); err != nil {
		return nil, fmt.Errorf("failed to parse har file: %w", err)
	}

	b.index.FileHash = fmt.Sprintf("%x", b.hash.Sum64())
	b.index.FileSize = hashReader.bytesRead
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

func (b *DefaultIndexBuilder) parseLog(decoder HARDecoder) error {
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

func (b *DefaultIndexBuilder) parseEntries(decoder HARDecoder) error {
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

// parseEntryMetadata selectively parses only metadata fields, skipping response bodies entirely
func (b *DefaultIndexBuilder) parseEntryMetadata(decoder HARDecoder, index int, startOffset int64) (*EntryMetadata, error) {
	metadata := &EntryMetadata{
		FileOffset: startOffset,
	}

	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('{') {
		return nil, fmt.Errorf("expected object delimiter, got %v", token)
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case keyStartedDateTime:
			var startTime string
			if err := decoder.Decode(&startTime); err != nil {
				return nil, err
			}
			if t, err := time.Parse(time.RFC3339, startTime); err == nil {
				metadata.Timestamp = t
			}

		case keyTime:
			if err := decoder.Decode(&metadata.Duration); err != nil {
				return nil, err
			}

		case keyRequest:
			if err := b.parseRequest(decoder, metadata); err != nil {
				return nil, err
			}

		case keyResponse:
			if err := b.parseResponse(decoder, metadata); err != nil {
				return nil, err
			}

		case keyPageRef:
			var pageRef string
			if err := decoder.Decode(&pageRef); err != nil {
				return nil, err
			}
			metadata.PageRef = b.index.Intern(pageRef)

		case keyServerIPAddress:
			var serverIP string
			if err := decoder.Decode(&serverIP); err != nil {
				return nil, err
			}
			metadata.ServerIP = b.index.Intern(serverIP)

		case keyConnection:
			var connection string
			if err := decoder.Decode(&connection); err != nil {
				return nil, err
			}
			metadata.Connection = b.index.Intern(connection)

		default:
			if err := helper.skipValue(decoder); err != nil {
				return nil, err
			}
		}
	}

	// consume closing brace
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (b *DefaultIndexBuilder) parseRequest(decoder HARDecoder, metadata *EntryMetadata) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected object delimiter")
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
		case keyMethod:
			var method string
			if err := decoder.Decode(&method); err != nil {
				return err
			}
			metadata.Method = b.index.Intern(method)

		case keyURL:
			var url string
			if err := decoder.Decode(&url); err != nil {
				return err
			}
			metadata.URL = b.index.Intern(url)

		case keyBodySize:
			var size int
			if err := decoder.Decode(&size); err != nil {
				return err
			}
			metadata.RequestSize = int64(size)

		default:
			if err := helper.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	// consume closing brace
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func (b *DefaultIndexBuilder) parseResponse(decoder HARDecoder, metadata *EntryMetadata) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected object delimiter")
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
		case keyStatus:
			if err := decoder.Decode(&metadata.StatusCode); err != nil {
				return err
			}

		case keyStatusText:
			var statusText string
			if err := decoder.Decode(&statusText); err != nil {
				return err
			}
			metadata.StatusText = b.index.Intern(statusText)

		case keyBodySize:
			var size int
			if err := decoder.Decode(&size); err != nil {
				return err
			}
			metadata.ResponseSize = int64(size)

		case keyContent:
			if err := b.parseResponseContent(decoder, metadata); err != nil {
				return err
			}

		default:
			if err := helper.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	// consume closing brace
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

// parseResponseContent extracts size and mimeType but SKIPS the body text entirely
func (b *DefaultIndexBuilder) parseResponseContent(decoder HARDecoder, metadata *EntryMetadata) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected object delimiter")
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
		case keySize:
			var size int
			if err := decoder.Decode(&size); err != nil {
				return err
			}
			metadata.BodySize = int64(size)

		case keyMimeType:
			var mimeType string
			if err := decoder.Decode(&mimeType); err != nil {
				return err
			}
			metadata.MimeType = b.index.Intern(mimeType)

		case keyText, keyEncoding:
			// skip without allocating the string value
			var discard json.RawMessage
			if err := decoder.Decode(&discard); err != nil {
				return err
			}
			discard = nil

		default:
			if err := helper.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	// consume closing brace
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func (b *DefaultIndexBuilder) AddEntry(offset int64, metadata *EntryMetadata) error {
	metadata.FileOffset = offset
	b.index.Entries = append(b.index.Entries, metadata)
	return nil
}

func (b *DefaultIndexBuilder) GetIndex() *Index {
	return b.index
}

type hashingReader struct {
	reader    io.Reader
	hash      *xxhash.Digest
	bytesRead int64
}

func (r *hashingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)
	if n > 0 && r.hash != nil {
		r.hash.Write(p[:n])
	}
	return n, err
}
