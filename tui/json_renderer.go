package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

// JSONRenderer handles rendering JSON with search highlighting and filtering
type JSONRenderer struct {
	searchEngine *JSONSearchEngine
	filtered     bool
	indent       string
	width        int
}

// NewJSONRenderer creates a new JSON renderer
func NewJSONRenderer(jsonContent string, width int) (*JSONRenderer, error) {
	engine, err := NewJSONSearchEngine(jsonContent)
	if err != nil {
		return nil, err
	}

	return &JSONRenderer{
		searchEngine: engine,
		filtered:     false,
		indent:       "  ",
		width:        width,
	}, nil
}

// SetSearch updates the search query
func (r *JSONRenderer) SetSearch(query string, keysOnly bool) {
	r.searchEngine.Search(query, keysOnly)
}

// ToggleFiltered toggles between filtered and full view
func (r *JSONRenderer) ToggleFiltered() {
	r.filtered = !r.filtered
}

// IsFiltered returns the current filter state
func (r *JSONRenderer) IsFiltered() bool {
	return r.filtered
}

// HasMatches returns true if there are search matches
func (r *JSONRenderer) HasMatches() bool {
	return len(r.searchEngine.matches) > 0
}

// GetMatchCount returns the number of matches
func (r *JSONRenderer) GetMatchCount() int {
	return len(r.searchEngine.matches)
}

// Render renders the JSON with highlighting and optional filtering
func (r *JSONRenderer) Render() string {
	var data interface{}

	if r.filtered && len(r.searchEngine.matches) > 0 {
		// Use filtered data
		filtered, err := r.searchEngine.FilterJSON(true)
		if err != nil || filtered == nil {
			data = r.searchEngine.parsed
		} else {
			data = filtered
		}
	} else {
		// Use full data
		data = r.searchEngine.parsed
	}

	// Render the JSON with proper indentation
	rendered := r.renderNode(data, "", 0)
	return rendered
}

// renderNode recursively renders a JSON node with highlighting
func (r *JSONRenderer) renderNode(node interface{}, path string, depth int) string {
	var out strings.Builder
	indent := strings.Repeat(r.indent, depth)

	switch v := node.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}

		out.WriteString("{\n")
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}

		for i, key := range keys {
			value := v[key]
			keyPath := key
			if path != "" {
				keyPath = path + "." + key
			}

			// Check if this key is matched or is a parent
			isMatched := r.searchEngine.IsPathMatched(keyPath)
			isParent := r.searchEngine.IsParentPath(keyPath)

			// Render the key with appropriate styling
			renderedKey := r.renderKey(key, isMatched, isParent, r.filtered)

			out.WriteString(indent + r.indent)
			out.WriteString(renderedKey)
			out.WriteString(": ")

			// Render the value
			valueStr := r.renderNode(value, keyPath, depth+1)
			out.WriteString(valueStr)

			if i < len(keys)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")
		}

		out.WriteString(indent + "}")

	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}

		out.WriteString("[\n")
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", path, i)

			out.WriteString(indent + r.indent)
			itemStr := r.renderNode(item, indexPath, depth+1)
			out.WriteString(itemStr)

			if i < len(v)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")
		}
		out.WriteString(indent + "]")

	case string:
		// Check if the value is matched
		isMatched := r.searchEngine.IsPathMatched(path)
		return r.renderValue(fmt.Sprintf("%q", v), isMatched)

	case float64:
		isMatched := r.searchEngine.IsPathMatched(path)
		// Format number nicely
		if v == float64(int64(v)) {
			return r.renderValue(fmt.Sprintf("%d", int64(v)), isMatched)
		}
		return r.renderValue(fmt.Sprintf("%g", v), isMatched)

	case bool:
		isMatched := r.searchEngine.IsPathMatched(path)
		return r.renderValue(fmt.Sprintf("%v", v), isMatched)

	case nil:
		isMatched := r.searchEngine.IsPathMatched(path)
		return r.renderValue("null", isMatched)

	default:
		// Fallback for any other type
		bytes, _ := json.Marshal(v)
		return string(bytes)
	}

	return out.String()
}

// renderKey renders a JSON key with appropriate styling
func (r *JSONRenderer) renderKey(key string, isMatched, isParent, inFilteredView bool) string {
	quotedKey := fmt.Sprintf("%q", key)

	// Create styles
	matchedStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)

	parentStyle := lipgloss.NewStyle().
		Foreground(RGBGrey).
		Faint(true)

	normalKeyStyle := SyntaxKeyStyle // Blue bold for normal keys

	// Apply appropriate style
	if isMatched {
		return matchedStyle.Render(quotedKey)
	} else if isParent && inFilteredView {
		return parentStyle.Render(quotedKey)
	} else {
		return normalKeyStyle.Render(quotedKey)
	}
}

// renderValue renders a JSON value with optional highlighting
func (r *JSONRenderer) renderValue(value string, isMatched bool) string {
	if isMatched && !r.searchEngine.matches[0].IsKey {
		// Highlight matched values
		matchedStyle := lipgloss.NewStyle().
			Background(RGBSubtlePink).
			Foreground(RGBPink).
			Bold(true)
		return matchedStyle.Render(value)
	}

	// Apply syntax highlighting for different value types
	if value == "true" || value == "false" {
		return SyntaxNumberStyle.Render(value) // Yellow for booleans
	} else if value == "null" {
		return lipgloss.NewStyle().Faint(true).Render(value)
	} else if strings.HasPrefix(value, "\"") {
		// String values - use default color
		return value
	} else {
		// Numbers - use yellow
		return SyntaxNumberStyle.Render(value)
	}
}

// RenderJSONWithSearch renders JSON content with search functionality
func RenderJSONWithSearch(content string, query string, keysOnly bool, filtered bool, width int) string {
	// Try to pretty print first
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(content), "", "  "); err != nil {
		// If pretty print fails, return original
		return content
	}

	// Create renderer
	renderer, err := NewJSONRenderer(prettyJSON.String(), width)
	if err != nil {
		// Fallback to syntax highlighted version without search
		return applySyntaxHighlightingToContent(prettyJSON.String(), false)
	}

	// Set search if provided
	if query != "" {
		renderer.SetSearch(query, keysOnly)
	}

	// Set filtered state
	if filtered {
		renderer.ToggleFiltered()
	}

	// Render
	return renderer.Render()
}

// Helper function to check if content is valid JSON
func isValidJSON(content string) bool {
	var js interface{}
	return json.Unmarshal([]byte(content), &js) == nil
}