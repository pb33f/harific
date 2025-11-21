package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// updateDetailContent updates the detail modal viewport content (e.g., after search changes)
func (m *HARViewModel) updateDetailContent() {
	modalWidth := int(float64(m.width) * 0.9)

	// Save current scroll position
	savedYOffset := m.detailViewport.YOffset

	var content string
	if m.activeModal == ModalRequestFull {
		content = m.formatRequestFullWithSearch(modalWidth - 4)
	} else {
		content = m.formatResponseFullWithSearch(modalWidth - 4)
	}

	m.detailViewport.SetContent(content)

	// Restore scroll position if valid
	if savedYOffset > 0 && savedYOffset < m.detailViewport.TotalLineCount() {
		m.detailViewport.SetYOffset(savedYOffset)
	}
}

// formatRequestFullWithSearch formats request with search applied
func (m *HARViewModel) formatRequestFullWithSearch(width int) string {
	if m.selectedEntry == nil {
		return "No request data"
	}

	sections := buildRequestSections(&m.selectedEntry.Request)

	opts := RenderOptions{
		Width:    width,
		Truncate: false,
	}

	// Use search-aware rendering if search is active
	if m.detailSearchState.active {
		// Don't apply syntax highlighting - let search renderer handle the JSON
		return renderSectionsWithSearch(sections, opts, m.detailSearchState)
	}

	// Apply syntax highlighting only when not searching
	sections = m.highlightBodyInSections(sections, m.selectedEntry.Request.Body.MIMEType)
	return renderSections(sections, opts)
}

// formatResponseFullWithSearch formats response with search applied
func (m *HARViewModel) formatResponseFullWithSearch(width int) string {
	if m.selectedEntry == nil {
		return "No response data"
	}

	sections := buildResponseSections(&m.selectedEntry.Response, &m.selectedEntry.Timings)

	opts := RenderOptions{
		Width:    width,
		Truncate: false,
	}

	// Use search-aware rendering if search is active
	if m.detailSearchState.active {
		// Don't apply syntax highlighting - let search renderer handle the JSON
		return renderSectionsWithSearch(sections, opts, m.detailSearchState)
	}

	// Apply syntax highlighting only when not searching
	sections = m.highlightBodyInSections(sections, m.selectedEntry.Response.Body.MIMEType)
	return renderSections(sections, opts)
}

// renderDetailSearchBar renders the search controls in the detail modal footer
func (m *HARViewModel) renderDetailSearchBar() string {
	searchState := m.detailSearchState
	var parts []string

	// Search input
	searchStyle := lipgloss.NewStyle().Foreground(RGBPink).Bold(true)
	parts = append(parts, searchStyle.Render("Search: ") + searchState.searchInput.View())

	// Key search checkbox
	checkbox := "[ ]"
	if searchState.keySearchOnly {
		checkbox = "[x]"
	}
	parts = append(parts, checkbox + " Keys Only")

	// Match count
	if searchState.query != "" && searchState.renderer != nil {
		matchCount := searchState.renderer.GetMatchCount()
		if matchCount > 0 {
			parts = append(parts, fmt.Sprintf("%d matches", matchCount))
		} else {
			parts = append(parts, "no matches")
		}
	}

	// Help
	helpParts := []string{"Tab: Toggle checkbox"}
	if searchState.filtered {
		helpParts = append(helpParts, "Enter: Show all")
	} else if len(searchState.matches) > 0 {
		helpParts = append(helpParts, "Enter: Filter")
	}
	helpParts = append(helpParts, "Esc: Exit search")

	helpStyle := lipgloss.NewStyle().Faint(true)
	parts = append(parts, helpStyle.Render(strings.Join(helpParts, " | ")))

	return strings.Join(parts, " | ")
}

// renderDetailModal renders the full request or response in a 90x90 modal
func (m *HARViewModel) renderDetailModal() string {
	modalWidth := int(float64(m.width) * 0.9)
	modalHeight := int(float64(m.height) * 0.9)

	// initialize viewport if needed
	if m.detailViewport.Width() == 0 {
		m.detailViewport = viewport.New(
			viewport.WithWidth(modalWidth-4),
			viewport.WithHeight(modalHeight-4),
		)
	}

	// prepare content based on modal type
	var content string
	var title string

	if m.activeModal == ModalRequestFull {
		title = "Request (Full View)"
		// Always use search-aware version - it handles both cases
		content = m.formatRequestFullWithSearch(modalWidth - 4)
	} else {
		title = "Response (Full View)"
		// Always use search-aware version - it handles both cases
		content = m.formatResponseFullWithSearch(modalWidth - 4)
	}

	m.detailViewport.SetContent(content)

	// modal styling
	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBBlue).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBBlue).
		Width(modalWidth - 4)

	helpStyle := lipgloss.NewStyle().
		Foreground(RGBGrey).
		Faint(true).
		Width(modalWidth - 4).
		Align(lipgloss.Center)

	var modal strings.Builder
	modal.WriteString(titleStyle.Render(title))
	modal.WriteString("\n")
	modal.WriteString(m.detailViewport.View())
	modal.WriteString("\n")

	// Show search controls if search is active, otherwise show normal help
	if m.detailSearchState.active {
		modal.WriteString(m.renderDetailSearchBar())
	} else {
		modal.WriteString(helpStyle.Render("↑/↓: Scroll | PgUp/PgDn: Page | Ctrl+F: Search | Esc: Close"))
	}

	return modalStyle.Render(modal.String())
}

// formatRequestFull formats request with full untruncated content and syntax highlighting
func (m *HARViewModel) formatRequestFull(width int) string {
	if m.selectedEntry == nil {
		return "No request data"
	}

	sections := buildRequestSections(&m.selectedEntry.Request)

	// apply syntax highlighting to body content
	sections = m.highlightBodyInSections(sections, m.selectedEntry.Request.Body.MIMEType)

	opts := RenderOptions{
		Width:    width,
		Truncate: false, // NO truncation in modal
	}

	return renderSections(sections, opts)
}

// formatResponseFull formats response with full untruncated content and syntax highlighting
func (m *HARViewModel) formatResponseFull(width int) string {
	if m.selectedEntry == nil {
		return "No response data"
	}

	sections := buildResponseSections(&m.selectedEntry.Response, &m.selectedEntry.Timings)

	// apply syntax highlighting to body content
	sections = m.highlightBodyInSections(sections, m.selectedEntry.Response.Body.MIMEType)

	opts := RenderOptions{
		Width:    width,
		Truncate: false, // NO truncation in modal
	}

	return renderSections(sections, opts)
}

// highlightBodyInSections applies syntax highlighting to body content in sections
func (m *HARViewModel) highlightBodyInSections(sections []Section, mimeType string) []Section {
	contentType := detectContentType(mimeType)

	// find and highlight body section
	for i, section := range sections {
		if section.Title == "Body" {
			for j, pair := range section.Pairs {
				if pair.Key == "Content" {
					content := pair.Value

					// pretty print JSON before highlighting
					if contentType == "json" {
						content = prettyPrintJSON(content)
					}

					// apply syntax highlighting
					if contentType != "plain" {
						content = applySyntaxHighlightingToContent(content, contentType == "yaml")
					}

					sections[i].Pairs[j].Value = content
				}
			}
		}
	}

	return sections
}

// detectContentType determines if content is JSON, YAML, or plain text
func detectContentType(mimeType string) string {
	lower := strings.ToLower(mimeType)

	if strings.Contains(lower, "json") {
		return "json"
	}
	if strings.Contains(lower, "yaml") || strings.Contains(lower, "yml") {
		return "yaml"
	}

	return "plain"
}

// prettyPrintJSON formats JSON with indentation
func prettyPrintJSON(jsonStr string) string {
	if jsonStr == "" {
		return jsonStr
	}

	// Parse and re-marshal with indentation
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	// Marshal with indentation
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return jsonStr
	}

	return string(bytes)
}

// applySyntaxHighlightingToContent applies line-by-line syntax highlighting
func applySyntaxHighlightingToContent(content string, isYAML bool) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	var highlighted strings.Builder

	for i, line := range lines {
		highlighted.WriteString(ApplySyntaxHighlightingToLine(line, isYAML))
		if i < len(lines)-1 {
			highlighted.WriteString("\n")
		}
	}

	return highlighted.String()
}

// handleDetailModalKeys handles key events when detail modal is open
func (m *HARViewModel) handleDetailModalKeys(key string) (bool, tea.Cmd) {
	if m.activeModal != ModalRequestFull && m.activeModal != ModalResponseFull {
		return false, nil
	}

	// If search is active, handle search-specific keys
	if m.detailSearchState.active {
		switch key {
		case "esc":
			// Exit search
			m.detailSearchState.Deactivate()
			m.updateDetailContent() // Update to show cleared search
			return true, nil

		case "tab":
			// Toggle between search input and checkbox
			m.detailSearchState.MoveCursor(1)
			if m.detailSearchState.cursor == 0 {
				return true, m.detailSearchState.searchInput.Focus()
			} else {
				m.detailSearchState.searchInput.Blur()
				return true, nil
			}

		case "enter", "return", " ", "space":
			if m.detailSearchState.cursor == 0 {
				// On input field - toggle filtered view if we have matches
				if len(m.detailSearchState.matches) > 0 {
					m.detailSearchState.ToggleFiltered()
					m.updateDetailContent()
				}
			} else {
				// On checkbox - toggle key search mode
				m.detailSearchState.ToggleKeySearchOnly()
				m.updateDetailContent()
			}
			return true, nil
		}

		// Let other keys fall through for search input handling
	}

	// Normal modal keys
	switch key {
	case "ctrl+f", "/":
		// Activate search and initialize with JSON content if available
		m.detailSearchState.Activate()

		// Get the JSON body content to search
		var jsonContent string
		if m.activeModal == ModalRequestFull && m.selectedEntry != nil {
			jsonContent = m.selectedEntry.Request.Body.Content
		} else if m.activeModal == ModalResponseFull && m.selectedEntry != nil {
			jsonContent = m.selectedEntry.Response.Body.Content
		}

		// Initialize the search state with the JSON content
		if jsonContent != "" && isValidJSON(jsonContent) {
			modalWidth := int(float64(m.width) * 0.9)
			m.detailSearchState.SetContent(jsonContent, modalWidth-4)
		}

		m.updateDetailContent()
		return true, m.detailSearchState.searchInput.Focus()

	case "esc":
		m.activeModal = ModalNone
		m.detailSearchState.Deactivate() // Also deactivate search when closing
		return true, nil

	case "up":
		m.detailViewport.LineUp(1)
		return true, nil

	case "down":
		m.detailViewport.LineDown(1)
		return true, nil

	case "pgup":
		m.detailViewport.ViewUp()
		return true, nil

	case "pgdown":
		m.detailViewport.ViewDown()
		return true, nil
	}

	return false, nil
}
