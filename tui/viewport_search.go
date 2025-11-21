package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/lipgloss/v2"
)

// ViewportSearchState tracks search state for a single viewport
type ViewportSearchState struct {
	active        bool
	query         string
	keySearchOnly bool
	matches       []JSONMatch
	filtered      bool
	searchInput   textinput.Model
	cursor        int  // 0 = input field, 1 = checkbox
	renderer      *JSONRenderer
}

// NewViewportSearchState creates a new viewport search state
func NewViewportSearchState() *ViewportSearchState {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Search JSON..."
	input.CharLimit = 100

	return &ViewportSearchState{
		active:        false,
		query:         "",
		keySearchOnly: true, // Default to key search only
		matches:       []JSONMatch{},
		filtered:      false,
		searchInput:   input,
		cursor:        0,
		renderer:      nil,
	}
}

// Activate activates the search UI
func (s *ViewportSearchState) Activate() {
	s.active = true
	s.searchInput.Focus()
	s.cursor = 0
}

// Deactivate deactivates the search UI
func (s *ViewportSearchState) Deactivate() {
	s.active = false
	s.searchInput.Blur()
	// Keep search results visible even when deactivated
}

// Clear clears the search state
func (s *ViewportSearchState) Clear() {
	s.query = ""
	s.matches = []JSONMatch{}
	s.filtered = false
	s.searchInput.SetValue("")
	s.renderer = nil
}

// SetContent updates the content being searched
func (s *ViewportSearchState) SetContent(jsonContent string, width int) error {
	if !isValidJSON(jsonContent) {
		// Not JSON content, disable search
		s.renderer = nil
		return fmt.Errorf("content is not valid JSON")
	}

	// Only create a new renderer if we don't have one already
	// This preserves the filtered state
	if s.renderer == nil {
		renderer, err := NewJSONRenderer(jsonContent, width)
		if err != nil {
			s.renderer = nil
			return err
		}
		s.renderer = renderer
	}

	// Re-apply current search if any
	if s.query != "" {
		s.performSearch()
	}

	// Sync filtered state with renderer
	if s.renderer != nil {
		s.renderer.filtered = s.filtered
	}

	return nil
}

// performSearch executes the search with current settings
func (s *ViewportSearchState) performSearch() {
	if s.renderer == nil {
		return
	}

	s.renderer.SetSearch(s.query, s.keySearchOnly)
	s.matches = s.renderer.searchEngine.matches

	// Sync filtered state with renderer
	s.renderer.filtered = s.filtered
}

// ToggleFiltered toggles between filtered and full view
func (s *ViewportSearchState) ToggleFiltered() {
	if s.renderer == nil || len(s.matches) == 0 {
		return
	}

	s.filtered = !s.filtered
	s.renderer.ToggleFiltered()
}

// ToggleKeySearchOnly toggles the key search mode
func (s *ViewportSearchState) ToggleKeySearchOnly() {
	s.keySearchOnly = !s.keySearchOnly
	// Re-run search with new mode
	if s.query != "" {
		s.performSearch()
	}
}

// MoveCursor moves the cursor between input and checkbox
func (s *ViewportSearchState) MoveCursor(direction int) {
	if direction > 0 {
		s.cursor++
		if s.cursor > 1 {
			s.cursor = 0
		}
	} else {
		s.cursor--
		if s.cursor < 0 {
			s.cursor = 1
		}
	}

	// Update focus
	if s.cursor == 0 {
		s.searchInput.Focus()
	} else {
		s.searchInput.Blur()
	}
}

// UpdateQuery updates the search query
func (s *ViewportSearchState) UpdateQuery(query string) {
	s.query = query
	s.searchInput.SetValue(query)
	if s.renderer != nil {
		s.performSearch()
	}
}

// RenderSearchPanel renders the search panel for the viewport
func (s *ViewportSearchState) RenderSearchPanel(width int) string {
	if !s.active {
		return ""
	}

	// Panel styling - make it a floating box
	panelStyle := lipgloss.NewStyle().
		Width(width - 4).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(RGBPink).
		Padding(0, 1).
		Background(lipgloss.Color("235")) // Dark background for visibility

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBPink)

	// Highlight style for focused element
	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)

	// Build content
	var content strings.Builder

	// Search label and input on same line
	content.WriteString(labelStyle.Render("Search: "))
	content.WriteString(s.searchInput.View())

	// Match count indicator
	if s.query != "" && s.renderer != nil {
		matchCount := s.renderer.GetMatchCount()
		countStyle := lipgloss.NewStyle().
			Foreground(RGBPink).
			Bold(true)

		if matchCount > 0 {
			countText := fmt.Sprintf(" (%d matches)", matchCount)
			content.WriteString(countStyle.Render(countText))
		} else {
			content.WriteString(countStyle.Render(" (no matches)"))
		}
	}

	content.WriteString("\n")

	// Checkbox for key search
	checkboxCursor := " "
	if s.cursor == 1 {
		checkboxCursor = ">"
	}

	checkbox := "[ ]"
	if s.keySearchOnly {
		checkbox = "[x]"
	}

	checkboxLine := fmt.Sprintf("%s %s Key Search Only", checkboxCursor, checkbox)

	if s.cursor == 1 {
		checkboxLine = highlightStyle.Render(checkboxLine)
	}

	content.WriteString(checkboxLine)

	// Help text
	content.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(RGBGrey).Faint(true)

	helpText := "Tab: Switch field | "
	if len(s.matches) > 0 {
		if s.filtered {
			helpText += "Enter/Space: Show all | "
		} else {
			helpText += "Enter/Space: Filter view | "
		}
	}
	helpText += "Esc: Close"

	content.WriteString(helpStyle.Render(helpText))

	return panelStyle.Render(content.String())
}

// GetRenderedContent returns the JSON content with search highlighting/filtering applied
func (s *ViewportSearchState) GetRenderedContent() string {
	if s.renderer == nil {
		return ""
	}

	return s.renderer.Render()
}

// HasJSONContent returns true if valid JSON content is loaded
func (s *ViewportSearchState) HasJSONContent() bool {
	return s.renderer != nil
}