package tui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONMatch represents a match found in JSON content
type JSONMatch struct {
	Path       string      // Full path to the match (e.g., "user.address.city")
	Key        string      // The actual key name
	Value      interface{} // The value at this path
	LineStart  int         // Starting line number in rendered output
	LineEnd    int         // Ending line number in rendered output
	IsKey      bool        // True if this match is a key match
	ParentPath string      // Path to the parent object/array
}

// JSONSearchEngine handles searching within JSON content
type JSONSearchEngine struct {
	content    string
	parsed     interface{}
	matches    []JSONMatch
	pathIndex  map[string]interface{} // Maps paths to values
	keyPaths   []string               // All key paths in the JSON
	searchMode SearchMode
}

// SearchMode defines how to search JSON content
type SearchMode int

const (
	SearchKeysOnly SearchMode = iota
	SearchKeysAndValues
)

// NewJSONSearchEngine creates a new JSON search engine
func NewJSONSearchEngine(jsonContent string) (*JSONSearchEngine, error) {
	engine := &JSONSearchEngine{
		content:   jsonContent,
		pathIndex: make(map[string]interface{}),
		keyPaths:  []string{},
		matches:   []JSONMatch{},
	}

	// Parse the JSON
	if err := json.Unmarshal([]byte(jsonContent), &engine.parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Build the path index
	engine.buildPathIndex(engine.parsed, "")

	return engine, nil
}

// buildPathIndex recursively builds an index of all paths in the JSON
func (e *JSONSearchEngine) buildPathIndex(data interface{}, parentPath string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			path := key
			if parentPath != "" {
				path = parentPath + "." + key
			}
			e.pathIndex[path] = value
			e.keyPaths = append(e.keyPaths, path)
			e.buildPathIndex(value, path)
		}

	case []interface{}:
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", parentPath, i)
			e.pathIndex[indexPath] = item
			e.buildPathIndex(item, indexPath)
		}
	}
}

// Search finds all matches for the given query
func (e *JSONSearchEngine) Search(query string, keysOnly bool) []JSONMatch {
	if query == "" {
		return []JSONMatch{}
	}

	e.matches = []JSONMatch{}
	e.searchMode = SearchKeysAndValues
	if keysOnly {
		e.searchMode = SearchKeysOnly
	}

	queryLower := strings.ToLower(query)

	// Search through all paths
	for path, value := range e.pathIndex {
		// Extract the key from the path
		parts := strings.Split(path, ".")
		key := parts[len(parts)-1]

		// Remove array indices from key for comparison
		keyClean := strings.Split(key, "[")[0]

		// Check if key matches
		if strings.Contains(strings.ToLower(keyClean), queryLower) {
			match := JSONMatch{
				Path:       path,
				Key:        keyClean,
				Value:      value,
				IsKey:      true,
				ParentPath: getParentPath(path),
			}
			e.matches = append(e.matches, match)
			continue
		}

		// If searching values too, check the value
		if !keysOnly {
			if matchesValue(value, queryLower) {
				match := JSONMatch{
					Path:       path,
					Key:        keyClean,
					Value:      value,
					IsKey:      false,
					ParentPath: getParentPath(path),
				}
				e.matches = append(e.matches, match)
			}
		}
	}

	return e.matches
}

// GetMatchedPaths returns all paths that have matches
func (e *JSONSearchEngine) GetMatchedPaths() []string {
	paths := []string{}
	seen := make(map[string]bool)

	for _, match := range e.matches {
		if !seen[match.Path] {
			paths = append(paths, match.Path)
			seen[match.Path] = true
		}

		// Also include all parent paths to maintain structure
		parentPaths := getAllParentPaths(match.Path)
		for _, p := range parentPaths {
			if !seen[p] {
				paths = append(paths, p)
				seen[p] = true
			}
		}
	}

	return paths
}

// IsPathMatched checks if a specific path has a match
func (e *JSONSearchEngine) IsPathMatched(path string) bool {
	for _, match := range e.matches {
		if match.Path == path {
			return true
		}
	}
	return false
}

// IsParentPath checks if a path is a parent of any matched path
func (e *JSONSearchEngine) IsParentPath(path string) bool {
	for _, match := range e.matches {
		if strings.HasPrefix(match.Path, path+".") || strings.HasPrefix(match.Path, path+"[") {
			return true
		}
	}
	return false
}

// Helper functions

func getParentPath(path string) string {
	// Handle array indices
	if idx := strings.LastIndex(path, "["); idx != -1 {
		return path[:idx]
	}

	// Handle object keys
	if idx := strings.LastIndex(path, "."); idx != -1 {
		return path[:idx]
	}

	return ""
}

func getAllParentPaths(path string) []string {
	var parents []string
	current := path

	for {
		parent := getParentPath(current)
		if parent == "" {
			break
		}
		parents = append(parents, parent)
		current = parent
	}

	return parents
}

func matchesValue(value interface{}, query string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), query)
	case float64:
		return strings.Contains(fmt.Sprintf("%v", v), query)
	case bool:
		return strings.Contains(fmt.Sprintf("%v", v), query)
	case nil:
		return strings.Contains("null", query)
	}
	return false
}

// FilterJSON returns a filtered version of the JSON containing only matched paths
func (e *JSONSearchEngine) FilterJSON(includeParents bool) (interface{}, error) {
	if len(e.matches) == 0 {
		return e.parsed, nil
	}

	matchedPaths := make(map[string]bool)
	for _, match := range e.matches {
		matchedPaths[match.Path] = true

		// Include all parent paths if requested
		if includeParents {
			for _, parent := range getAllParentPaths(match.Path) {
				matchedPaths[parent] = true
			}
		}
	}

	// Rebuild the JSON with only matched paths
	return e.filterNode(e.parsed, "", matchedPaths), nil
}

// filterNode recursively filters a JSON node based on matched paths
func (e *JSONSearchEngine) filterNode(node interface{}, currentPath string, matchedPaths map[string]bool) interface{} {
	switch v := node.(type) {
	case map[string]interface{}:
		filtered := make(map[string]interface{})

		for key, value := range v {
			path := key
			if currentPath != "" {
				path = currentPath + "." + key
			}

			// Include if this path or any child path is matched
			if e.shouldIncludePath(path, matchedPaths) {
				filtered[key] = e.filterNode(value, path, matchedPaths)
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil

	case []interface{}:
		var filtered []interface{}

		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)

			if e.shouldIncludePath(indexPath, matchedPaths) {
				filteredItem := e.filterNode(item, indexPath, matchedPaths)
				if filteredItem != nil {
					filtered = append(filtered, filteredItem)
				}
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
		return nil

	default:
		// Primitive values are included if their path is matched
		if matchedPaths[currentPath] {
			return v
		}
		return nil
	}
}

// shouldIncludePath checks if a path should be included in filtered output
func (e *JSONSearchEngine) shouldIncludePath(path string, matchedPaths map[string]bool) bool {
	// Include if this exact path is matched
	if matchedPaths[path] {
		return true
	}

	// Include if any child path is matched
	for matchPath := range matchedPaths {
		if strings.HasPrefix(matchPath, path+".") || strings.HasPrefix(matchPath, path+"[") {
			return true
		}
	}

	return false
}