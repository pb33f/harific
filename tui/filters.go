package tui

import (
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
