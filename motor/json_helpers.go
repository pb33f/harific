package motor

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonHelper struct{}

var helper = &jsonHelper{}

func (h *jsonHelper) skipValue(decoder *json.Decoder) error {
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

func (h *jsonHelper) skipObject(decoder *json.Decoder) error {
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

func (h *jsonHelper) skipArray(decoder *json.Decoder) error {
	for decoder.More() {
		if err := h.skipValue(decoder); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func (h *jsonHelper) navigateToEntries(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		return err
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		if key, ok := token.(string); ok && key == keyLog {
			if _, err := decoder.Token(); err != nil {
				return err
			}

			for decoder.More() {
				token, err := decoder.Token()
				if err != nil {
					return err
				}

				if key, ok := token.(string); ok && key == keyEntries {
					if _, err := decoder.Token(); err != nil {
						return err
					}
					return nil
				}

				if err := h.skipValue(decoder); err != nil {
					return err
				}
			}
		} else {
			if err := h.skipValue(decoder); err != nil {
				return err
			}
		}
	}

	return fmt.Errorf("entries array not found")
}

func newHARDecoder(r io.Reader) *json.Decoder {
	d := json.NewDecoder(r)
	d.UseNumber()
	return d
}
