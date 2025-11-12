package motor

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStdlibTokenParser_Init(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"test": "value"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if parser.decoder == nil {
		t.Error("expected non-nil decoder")
	}

	if parser.depth != 0 {
		t.Errorf("expected depth 0, got %d", parser.depth)
	}
}

func TestStdlibTokenParser_InitWithOffset(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"test": "value"}`
	reader := strings.NewReader(jsonData)

	// init with offset (should seek to that position)
	err := parser.Init(reader, 5)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if parser.pos != 5 {
		t.Errorf("expected pos 5, got %d", parser.pos)
	}
}

func TestStdlibTokenParser_NextToken(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"test": "value"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// read opening brace
	token, err := parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != json.Delim('{') {
		t.Errorf("expected opening brace, got %v", token)
	}

	if parser.depth != 1 {
		t.Errorf("expected depth 1, got %d", parser.depth)
	}

	// read key
	token, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != "test" {
		t.Errorf("expected 'test', got %v", token)
	}

	// read value
	token, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != "value" {
		t.Errorf("expected 'value', got %v", token)
	}

	// read closing brace
	token, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != json.Delim('}') {
		t.Errorf("expected closing brace, got %v", token)
	}

	if parser.depth != 0 {
		t.Errorf("expected depth 0, got %d", parser.depth)
	}
}

func TestStdlibTokenParser_Skip(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"skip": {"nested": "value"}, "keep": "test"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// read opening brace
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// read "skip" key
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// skip the nested object
	err = parser.Skip()
	if err != nil {
		t.Fatalf("skip failed: %v", err)
	}

	// next token should be "keep"
	token, err := parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != "keep" {
		t.Errorf("expected 'keep' after skip, got %v", token)
	}
}

func TestStdlibTokenParser_SkipArray(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"array": [1, 2, 3], "after": "value"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// read opening brace
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// read "array" key
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// skip the array
	err = parser.Skip()
	if err != nil {
		t.Fatalf("skip failed: %v", err)
	}

	// next token should be "after"
	token, err := parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != "after" {
		t.Errorf("expected 'after' after skip, got %v", token)
	}
}

func TestStdlibTokenParser_Decode(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"name": "test", "value": 123}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// decode entire object
	var result map[string]interface{}
	err = parser.Decode(&result)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("expected name 'test', got %v", result["name"])
	}

	if result["value"].(json.Number).String() != "123" {
		t.Errorf("expected value 123, got %v", result["value"])
	}
}

func TestStdlibTokenParser_NavigateTo(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"log": {"version": "1.2", "entries": []}}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// navigate to "log" -> "entries"
	err = parser.NavigateTo([]string{"log", "entries"})
	if err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	// should be at the start of entries array
	// (currently pointing at the value after "entries" key)
}

func TestStdlibTokenParser_Position(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"test": "value"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 10)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	pos := parser.Position()
	if pos != 10 {
		t.Errorf("expected position 10, got %d", pos)
	}
}

func TestStdlibTokenParser_Depth(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"outer": {"inner": "value"}}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if parser.Depth() != 0 {
		t.Errorf("expected initial depth 0, got %d", parser.Depth())
	}

	// read opening brace
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if parser.Depth() != 1 {
		t.Errorf("expected depth 1 after opening brace, got %d", parser.Depth())
	}

	// read "outer" key
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// read inner opening brace
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if parser.Depth() != 2 {
		t.Errorf("expected depth 2 after second opening brace, got %d", parser.Depth())
	}
}

func TestStdlibTokenParser_SkipPrimitive(t *testing.T) {
	parser := NewStdlibTokenParser()

	jsonData := `{"a": 123, "b": "test"}`
	reader := strings.NewReader(jsonData)

	err := parser.Init(reader, 0)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// read opening brace
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// read "a" key
	_, err = parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	// skip the number value (primitive)
	err = parser.Skip()
	if err != nil {
		t.Fatalf("skip failed: %v", err)
	}

	// next token should be "b"
	token, err := parser.NextToken()
	if err != nil {
		t.Fatalf("next token failed: %v", err)
	}

	if token != "b" {
		t.Errorf("expected 'b' after skip, got %v", token)
	}
}
