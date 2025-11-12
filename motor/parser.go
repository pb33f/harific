package motor

import (
	"encoding/json"
	"fmt"
	"io"
)

type StdlibTokenParser struct {
	reader  io.ReadSeeker
	decoder *json.Decoder
	depth   int
	pos     int64
}

func NewStdlibTokenParser() *StdlibTokenParser {
	return &StdlibTokenParser{}
}

func (p *StdlibTokenParser) Init(reader io.ReadSeeker, offset int64) error {
	p.reader = reader
	p.depth = 0

	if offset > 0 {
		_, err := reader.Seek(offset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek to offset %d: %w", offset, err)
		}
		p.pos = offset
	}

	p.decoder = json.NewDecoder(reader)
	p.decoder.UseNumber()
	return nil
}

func (p *StdlibTokenParser) NavigateTo(path []string) error {
	if len(path) == 0 {
		return nil
	}

	for _, key := range path {
		found := false
		for {
			token, err := p.decoder.Token()
			if err != nil {
				if err == io.EOF {
					return fmt.Errorf("path not found: %v", path)
				}
				return err
			}

			switch token {
			case json.Delim('{'), json.Delim('['):
				p.depth++
			case json.Delim('}'), json.Delim(']'):
				p.depth--
			}

			if str, ok := token.(string); ok && str == key {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("key %s not found in path %v", key, path)
		}
	}

	return nil
}

func (p *StdlibTokenParser) NextToken() (json.Token, error) {
	token, err := p.decoder.Token()
	if err != nil {
		return nil, err
	}

	switch token {
	case json.Delim('{'), json.Delim('['):
		p.depth++
	case json.Delim('}'), json.Delim(']'):
		p.depth--
	}

	return token, nil
}

func (p *StdlibTokenParser) Skip() error {
	startDepth := p.depth

	token, err := p.NextToken()
	if err != nil {
		return err
	}

	switch token {
	case json.Delim('{'), json.Delim('['):
		for p.depth > startDepth {
			_, err := p.NextToken()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *StdlibTokenParser) Decode(v interface{}) error {
	return p.decoder.Decode(v)
}

func (p *StdlibTokenParser) Position() int64 {
	return p.pos
}

func (p *StdlibTokenParser) Depth() int {
	return p.depth
}
