package tui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

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
		content = m.formatRequestFull(modalWidth - 4)
	} else {
		title = "Response (Full View)"
		content = m.formatResponseFull(modalWidth - 4)
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
	modal.WriteString(helpStyle.Render("↑/↓: Scroll | PgUp/PgDn: Page | Esc: Close"))

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

	var buf bytes.Buffer
	err := json.Indent(&buf, []byte(jsonStr), "", "  ")
	if err != nil {
		// if indent fails, return original
		return jsonStr
	}

	return buf.String()
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

	switch key {
	case "esc":
		m.activeModal = ModalNone
		return true, nil

	case "up", "k":
		m.detailViewport.LineUp(1)
		return true, nil

	case "down", "j":
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
