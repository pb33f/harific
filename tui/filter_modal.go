package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

var fileTypeCategories = []string{"Graphics", "JS", "CSS", "Fonts", "Markup", "All Files"}

func (m *HARViewModel) renderFilterModal() string {
	// Fixed modal width for consistent appearance
	modalWidth := 30

	modalStyle := lipgloss.NewStyle().
		Width(modalWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBBlue).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBBlue)

	highlightStyle := lipgloss.NewStyle().
		Background(RGBSubtlePink).
		Foreground(RGBPink).
		Bold(true)

	var content strings.Builder

	content.WriteString(titleStyle.Render("File Type Filters"))
	content.WriteString("\n\n")

	// Always use single column layout for clean appearance
	for i, category := range fileTypeCategories {
		cursor := " "
		if m.filterCursor == i {
			cursor = ">"
		}

		checkbox := "[x]"
		if !m.filterCheckboxes[i] {
			checkbox = "[ ]"
		}

		line := fmt.Sprintf("%s %s %-12s", cursor, checkbox, category)

		if m.filterCursor == i {
			line = highlightStyle.Render(line)
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	// add Reset option
	content.WriteString("\n")
	resetLine := " [ ] Reset All Filters"
	if m.filterCursor == len(fileTypeCategories) {
		resetLine = highlightStyle.Render("> [*] Reset All Filters")
	}
	content.WriteString(resetLine)

	// help text
	content.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().Foreground(RGBGrey).Faint(true)
	content.WriteString(helpStyle.Render("↑/↓: Navigate | Space: Toggle | R: Reset | Esc: Close"))

	return modalStyle.Render(content.String())
}

func (m *HARViewModel) toggleFilterCheckbox() {
	if m.filterCursor < len(fileTypeCategories) {
		m.filterCheckboxes[m.filterCursor] = !m.filterCheckboxes[m.filterCursor]
		m.updateFileTypeFilter()
		m.applyFilters()
	} else if m.filterCursor == len(fileTypeCategories) {
		// reset option selected
		m.resetFileTypeFilters()
	}
}

func (m *HARViewModel) updateFileTypeFilter() {
	for i, category := range fileTypeCategories {
		m.fileTypeFilter.ToggleCategory(category, m.filterCheckboxes[i])
	}
}

func (m *HARViewModel) resetFileTypeFilters() {
	m.filterCheckboxes = [6]bool{true, true, true, true, true, true}
	m.fileTypeFilter.Clear()
	m.applyFilters()
}

func (m *HARViewModel) handleFilterModalKeys(key string) (bool, tea.Cmd) {
	if m.activeModal != ModalFileTypeFilter {
		return false, nil
	}

	switch key {
	case "esc", "f":
		m.activeModal = ModalNone
		return true, nil

	case "r":
		m.resetFileTypeFilters()
		return true, nil

	case "up":
		m.filterCursor--
		if m.filterCursor < 0 {
			m.filterCursor = len(fileTypeCategories) // wrap to Reset option
		}
		return true, nil

	case "down":
		m.filterCursor++
		if m.filterCursor > len(fileTypeCategories) {
			m.filterCursor = 0 // wrap to first checkbox
		}
		return true, nil

	case " ", "space", "enter":
		m.toggleFilterCheckbox()
		return true, nil
	}

	return false, nil
}
