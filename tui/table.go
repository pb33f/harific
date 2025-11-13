package tui

import (
	"fmt"
	"net/url"
	"time"

	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/pb33f/braid/motor"
)

func (m *HARViewModel) buildTableRows() {
	rows := make([]table.Row, 0, len(m.allEntries))

	for _, entry := range m.allEntries {
		row := formatEntryRow(entry, m.width)
		rows = append(rows, row)
	}

	m.rows = rows
}

func formatEntryRow(entry *motor.EntryMetadata, terminalWidth int) table.Row {
	method := formatMethod(entry.Method)
	urlPath := formatURL(entry.URL, terminalWidth)
	status := formatStatus(entry.StatusCode, entry.StatusText)
	duration := formatDuration(entry.Duration)

	return table.Row{method, urlPath, status, duration}
}

func formatMethod(method string) string {
	if method == "" {
		method = "GET"
	}

	if len(method) > 7 {
		return method[:7]
	}

	return method
}

func formatURL(fullURL string, terminalWidth int) string {
	if fullURL == "" {
		return "/"
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		if len(fullURL) > maxURLDisplayLength {
			return fullURL[:maxURLDisplayLength-3] + "..."
		}
		return fullURL
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	if u.RawQuery != "" {
		path = path + "?" + u.RawQuery
	}

	availableWidth := terminalWidth - methodColumnWidth - statusColumnWidth - durationColumnWidth - 10
	if availableWidth < minURLColumnWidth {
		availableWidth = minURLColumnWidth
	}
	if availableWidth > maxURLColumnWidth {
		availableWidth = maxURLColumnWidth
	}

	if len(path) > availableWidth {
		return path[:availableWidth-3] + "..."
	}

	return path
}

func formatStatus(code int, text string) string {
	if code == 0 {
		return "---"
	}

	if text != "" {
		status := fmt.Sprintf("%d %s", code, text)
		if len(status) > 10 {
			return fmt.Sprintf("%d", code)
		}
		return status
	}

	return fmt.Sprintf("%d", code)
}

func formatDuration(durationMs float64) string {
	if durationMs == 0 {
		return "---"
	}

	d := time.Duration(durationMs * float64(time.Millisecond))

	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dμs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		seconds := float64(d.Milliseconds()) / 1000.0
		return fmt.Sprintf("%.1fs", seconds)
	default:
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) - (minutes * 60)
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}