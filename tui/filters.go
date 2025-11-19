package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/pb33f/braid/motor"
)

// EntryFilter defines a filter that can show/hide entries
type EntryFilter interface {
	ShouldShow(index int, metadata *motor.EntryMetadata) bool
	IsActive() bool
}

// SearchFilter filters entries based on search results
type SearchFilter struct {
	matches     map[int]struct{}
	hasSearched bool // true if a search has been executed (even if 0 results)
}

// NewSearchFilter creates a new search filter
func NewSearchFilter() *SearchFilter {
	return &SearchFilter{
		matches: make(map[int]struct{}),
	}
}

// ShouldShow returns true if the entry index is in the match set
func (f *SearchFilter) ShouldShow(index int, metadata *motor.EntryMetadata) bool {
	_, found := f.matches[index]
	return found
}

// IsActive returns true if a search has been executed
func (f *SearchFilter) IsActive() bool {
	return f.hasSearched
}

// SetSearched marks the filter as having executed a search
func (f *SearchFilter) SetSearched(searched bool) {
	f.hasSearched = searched
}

// AddMatch adds an entry index to the match set
func (f *SearchFilter) AddMatch(index int) {
	f.matches[index] = struct{}{}
}

// Clear removes all matches and marks filter as inactive
func (f *SearchFilter) Clear() {
	f.matches = make(map[int]struct{})
	f.hasSearched = false
}

// MatchCount returns the number of matches
func (f *SearchFilter) MatchCount() int {
	return len(f.matches)
}

// FilterChain combines multiple filters
type FilterChain struct {
	filters []EntryFilter
}

// NewFilterChain creates a new filter chain
func NewFilterChain() *FilterChain {
	return &FilterChain{
		filters: make([]EntryFilter, 0, 4), // pre-allocate for common case
	}
}

// Add adds a filter to the chain
func (fc *FilterChain) Add(filter EntryFilter) {
	if filter != nil && filter.IsActive() {
		fc.filters = append(fc.filters, filter)
	}
}

// Clear removes all filters
func (fc *FilterChain) Clear() {
	fc.filters = fc.filters[:0]
}

// HasActiveFilters returns true if any filters are active
func (fc *FilterChain) HasActiveFilters() bool {
	return len(fc.filters) > 0
}

// BuildFilteredRows applies all active filters to build a filtered row set
func (fc *FilterChain) BuildFilteredRows(allEntries []*motor.EntryMetadata, allRows []table.Row) []table.Row {
	if !fc.HasActiveFilters() {
		return allRows
	}

	filtered := make([]table.Row, 0, len(allRows))

	for i, row := range allRows {
		if i >= len(allEntries) {
			continue
		}

		// entry must pass ALL filters
		passesAll := true
		for _, filter := range fc.filters {
			if !filter.ShouldShow(i, allEntries[i]) {
				passesAll = false
				break
			}
		}

		if passesAll {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

// FileTypeFilter filters entries based on file extensions
type FileTypeFilter struct {
	excludedCategories map[string]bool
}

// file extension to category mapping
var extensionToCategory = map[string]string{
	".png": "Graphics", ".gif": "Graphics", ".webp": "Graphics",
	".jpg": "Graphics", ".jpeg": "Graphics", ".svg": "Graphics", ".ico": "Graphics",
	".js": "JS", ".jsx": "JS", ".ts": "JS", ".tsx": "JS", ".mjs": "JS",
	".css": "CSS", ".scss": "CSS", ".sass": "CSS", ".less": "CSS",
	".woff": "Fonts", ".woff2": "Fonts", ".ttf": "Fonts", ".eot": "Fonts", ".otf": "Fonts",
	".html": "Markup", ".htm": "Markup", ".xml": "Markup", ".yaml": "Markup", ".yml": "Markup",
}

// NewFileTypeFilter creates a new file type filter
func NewFileTypeFilter() *FileTypeFilter {
	return &FileTypeFilter{
		excludedCategories: make(map[string]bool),
	}
}

// ShouldShow returns true if the entry URL is not an excluded file type
func (f *FileTypeFilter) ShouldShow(index int, metadata *motor.EntryMetadata) bool {
	// extract extension from URL
	ext := strings.ToLower(filepath.Ext(metadata.URL))
	if ext == "" {
		return true // no extension, likely an API endpoint
	}

	// check if extension belongs to a known category
	category := extensionToCategory[ext]
	if category == "" {
		// unknown extension - falls under "All Files" category
		category = "All Files"
	}

	return !f.excludedCategories[category]
}

// IsActive returns true if any categories are excluded
func (f *FileTypeFilter) IsActive() bool {
	return len(f.excludedCategories) > 0
}

// ExcludeCategory adds a category to the exclusion list
func (f *FileTypeFilter) ExcludeCategory(category string) {
	f.excludedCategories[category] = true
}

// IncludeCategory removes a category from the exclusion list
func (f *FileTypeFilter) IncludeCategory(category string) {
	delete(f.excludedCategories, category)
}

// Clear removes all exclusions
func (f *FileTypeFilter) Clear() {
	f.excludedCategories = make(map[string]bool)
}

// ToggleCategory toggles a category's exclusion state
func (f *FileTypeFilter) ToggleCategory(category string, include bool) {
	if include {
		f.IncludeCategory(category)
	} else {
		f.ExcludeCategory(category)
	}
}

