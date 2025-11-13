package motor

import (
	"encoding/json"
	"io"
)

type jsonHelper struct{}

var helper = &jsonHelper{}

// StdlibDecoder wraps encoding/json.Decoder to implement HARDecoder interface
// to swap to sonic: create SonicDecoder implementing HARDecoder, update newHARDecoder()
type StdlibDecoder struct {
	decoder *json.Decoder
}

func (s *StdlibDecoder) Token() (json.Token, error) {
	return s.decoder.Token()
}

func (s *StdlibDecoder) Decode(v interface{}) error {
	return s.decoder.Decode(v)
}

func (s *StdlibDecoder) More() bool {
	return s.decoder.More()
}

func (s *StdlibDecoder) InputOffset() int64 {
	return s.decoder.InputOffset()
}

func (h *jsonHelper) skipValue(decoder HARDecoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}

	switch token {
	case json.Delim('{'):
		return h.skipObject(decoder)
	case json.Delim('['):
		return h.skipArray(decoder)
	}

	return nil
}

func (h *jsonHelper) skipObject(decoder HARDecoder) error {
	for decoder.More() {
		if _, err := decoder.Token(); err != nil {
			return err
		}
		if err := h.skipValue(decoder); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func (h *jsonHelper) skipArray(decoder HARDecoder) error {
	for decoder.More() {
		if err := h.skipValue(decoder); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func newHARDecoder(r io.Reader) HARDecoder {
	d := json.NewDecoder(r)
	d.UseNumber()
	return &StdlibDecoder{decoder: d}
}
