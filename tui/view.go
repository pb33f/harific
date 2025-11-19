package tui

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss/v2"
)

const (
    maxBodyDisplayLength = 5000
    maxURLDisplayLength  = 50
)

func truncateBody(content string, maxLen int) string {
    if len(content) <= maxLen {
        return content
    }
    return content[:maxLen] + "\n...[truncated]"
}

func (m *HARViewModel) render() string {
    if m.err != nil {
        return m.renderError()
    }

    switch m.viewMode {
    case ViewModeTableWithSplit:
        return m.renderSplitView()
    case ViewModeTableWithSearch:
        return m.renderSearchView()
    default:
        return m.renderTableView()
    }
}

func (m *HARViewModel) renderTableView() string {
    var builder strings.Builder

    builder.WriteString(m.renderTitle())
    builder.WriteString("\n")

    // post-process table view to add colorization (vacuum pattern)
    tableView := m.table.View()
    colorizedTable := ColorizeHARTableOutput(tableView, m.table.Cursor(), m.rows)
    builder.WriteString(colorizedTable)

    builder.WriteString("\n")
    builder.WriteString(m.renderStatusBar())

    return builder.String()
}

func (m *HARViewModel) renderSplitView() string {
    var builder strings.Builder

    builder.WriteString(m.renderTitle())
    builder.WriteString("\n")

    // post-process table view to add colorization (vacuum pattern)
    tableView := m.table.View()
    colorizedTable := ColorizeHARTableOutput(tableView, m.table.Cursor(), m.rows)
    builder.WriteString(colorizedTable)

    builder.WriteString("\n")
    builder.WriteString(m.renderSplitPanel())
    builder.WriteString("\n")
    builder.WriteString(m.renderStatusBar())

    return builder.String()
}

func (m *HARViewModel) renderSearchView() string {
    var builder strings.Builder

    builder.WriteString(m.renderTitle())
    builder.WriteString("\n")

    // use cached colorized table to avoid re-rendering on every keystroke
    if m.cachedColorizedTable != "" {
        builder.WriteString(m.cachedColorizedTable)
    } else {
        // fallback to dynamic rendering if cache is empty
        tableView := m.table.View()
        colorizedTable := ColorizeHARTableOutput(tableView, m.table.Cursor(), m.rows)
        builder.WriteString(colorizedTable)
    }

    builder.WriteString("\n")
    builder.WriteString(m.renderSearchPanel())
    builder.WriteString("\n")
    builder.WriteString(m.renderStatusBar())

    return builder.String()
}

func (m *HARViewModel) renderTitle() string {
    title := fmt.Sprintf("HARific: %s | ", m.fileName)
    titleStyle := lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        Padding(0, 1).
        Width(m.width).BorderForeground(RGBBlue).BorderTop(false).BorderLeft(false).BorderRight(false).BorderBottom(true)

    titleTextStyle := lipgloss.NewStyle().
        Bold(true)

    titleText := titleTextStyle.Render(title)

    entryCount := fmt.Sprintf("(%d entries", len(m.allEntries))
    if m.indexingTime > 0 {
        entryCount += fmt.Sprintf(", loaded in %v", m.indexingTime.Round(time.Millisecond))
    }
    entryCount += ")"

    countStyle := lipgloss.NewStyle().
        Faint(true)

    return titleStyle.Render(titleText + countStyle.Render(entryCount))
}

func (m *HARViewModel) renderStatusBar() string {
    var parts []string

    if m.viewMode == ViewModeTable {
        parts = append(parts, "↑/↓: Navigate")
        parts = append(parts, "Enter: View Details")
        parts = append(parts, "s: Search")
    } else if m.viewMode == ViewModeTableWithSearch {
        parts = append(parts, "↑/↓: Navigate")
        parts = append(parts, "←/→: Jump to Input")
        parts = append(parts, "Space: Toggle")
        parts = append(parts, "Enter: Search")
        parts = append(parts, "Esc: Review Results")
    } else if m.viewMode == ViewModeTableFiltered {
        parts = append(parts, "↑/↓: Navigate")
        parts = append(parts, "Enter: View Details")
        parts = append(parts, "s: Search")
        parts = append(parts, "Esc: Clear Filters")
    } else {
        parts = append(parts, "↑/↓: Scroll")
        parts = append(parts, "Tab: Switch Panel")
        parts = append(parts, "Esc: Close Details")
    }

    parts = append(parts, "q: Quit")

    if m.selectedIndex < len(m.allEntries) {
        info := fmt.Sprintf("Entry %d/%d", m.selectedIndex+1, len(m.allEntries))
        parts = append(parts, info)
    }

    // Add focus indicator at the end when in split view
    if m.viewMode == ViewModeTableWithSplit {
        if m.focusedViewport == ViewportFocusRequest {
            parts = append(parts, "[Request]")
        } else {
            parts = append(parts, "[Response]")
        }
    }

    statusStyle := lipgloss.NewStyle().Faint(true)
    return statusStyle.Render(strings.Join(parts, " | "))
}

func (m *HARViewModel) renderSplitPanel() string {
    if m.selectedEntry == nil {
        return m.renderEmptyPanel()
    }

    panelWidth, panelHeight := m.calculatePanelDimensions()

    baseStyle := lipgloss.NewStyle().
        Width(panelWidth).
        Height(panelHeight).
        BorderStyle(lipgloss.NormalBorder())

    focusedBorderStyle := baseStyle.BorderForeground(RGBBlue)
    unfocusedBorderStyle := baseStyle.BorderForeground(lipgloss.Color("240"))

    leftBorderStyle := unfocusedBorderStyle
    rightBorderStyle := unfocusedBorderStyle

    if m.focusedViewport == ViewportFocusRequest {
        leftBorderStyle = focusedBorderStyle
    } else {
        rightBorderStyle = focusedBorderStyle
    }

    leftPanel := leftBorderStyle.Render(m.requestViewport.View())
    rightPanel := rightBorderStyle.Render(m.responseViewport.View())

    return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// renderSearchPanel creates the search input panel with pink border styling.
// The panel takes 30% of the vertical space at the bottom of the screen.
func (m *HARViewModel) renderSearchPanel() string {
    searchStyle := lipgloss.NewStyle().
        Width(m.width).
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(RGBPink).
        Padding(0, 1)

    labelStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(RGBPink)

    // background highlight style for focused checkbox (same as table row)
    highlightStyle := lipgloss.NewStyle().
        Background(RGBSubtlePink).
        Foreground(RGBPink).
        Bold(true)

    var content strings.Builder

    // Search label with spinner after when actively searching
    content.WriteString(labelStyle.Render("Search:"))
    if m.isSearching {
        content.WriteString(" ")
        content.WriteString(m.searchSpinner.View())
    }
    content.WriteString("\n")
    content.WriteString(m.searchInput.View())
    content.WriteString("\n")

    // Render checkboxes
    checkboxes := []struct {
        label   string
        checked bool
        index   int
    }{
        {"Response Bodies", m.searchOptions[0], searchCursorOpt1},
        {"Regex Mode", m.searchOptions[1], searchCursorOpt2},
        {"All Matches", m.searchOptions[2], searchCursorOpt3},
        {"Live Search", m.searchOptions[3], searchCursorOpt4},
    }

    for i, cb := range checkboxes {
        cursor := " "
        if m.searchCursor == cb.index {
            cursor = ">"
        }

        checkbox := "[ ]"
        if cb.checked {
            checkbox = "[x]"
        }

        line := fmt.Sprintf("%s %s %s", cursor, checkbox, cb.label)

        // apply background highlight if this checkbox is focused
        if m.searchCursor == cb.index {
            line = highlightStyle.Render(line)
        }

        content.WriteString(line)
        // don't add newline after last checkbox
        if i < len(checkboxes)-1 {
            content.WriteString("\n")
        }
    }

    return searchStyle.Render(content.String())
}

func (m *HARViewModel) renderEmptyPanel() string {
    emptyStyle := lipgloss.NewStyle().
        Faint(true).
        Align(lipgloss.Center, lipgloss.Center).
        Width(m.width).
        Height(m.height / 2)

    return emptyStyle.Render("No entry selected")
}

func (m *HARViewModel) formatRequest() string {
    if m.selectedEntry == nil {
        return "No request data"
    }

    sections := buildRequestSections(&m.selectedEntry.Request)

    opts := RenderOptions{
        Width:    m.requestViewport.Width(),
        Truncate: true,
    }

    return renderSections(sections, opts)
}

func (m *HARViewModel) formatResponse() string {
    if m.selectedEntry == nil {
        return "No response data"
    }

    sections := buildResponseSections(&m.selectedEntry.Response, &m.selectedEntry.Timings)

    opts := RenderOptions{
        Width:    m.responseViewport.Width(),
        Truncate: true,
    }

    return renderSections(sections, opts)
}

func (m *HARViewModel) renderError() string {
    errorStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("9")).
        Bold(true)

    return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
}

func (m *HARViewModel) updateViewportContent() {
    if m.selectedEntry == nil {
        return
    }

    requestContent := m.formatRequest()
    m.requestViewport.SetContent(requestContent)

    responseContent := m.formatResponse()
    m.responseViewport.SetContent(responseContent)
}
