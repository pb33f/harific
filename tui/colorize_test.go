package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/v2/table"
)

func TestTableStructure(t *testing.T) {
	// Create a simple table with a few rows
	columns := []table.Column{
		{Title: "Method", Width: 10},
		{Title: "URL", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Duration", Width: 10},
	}

	rows := []table.Row{
		{"GET", "/api/users", "200", "100ms"},
		{"POST", "/api/orders", "201", "250ms"},
		{"DELETE", "/api/items/123", "404", "50ms"},
		{"PUT", "/api/products", "500", "1.2s"},
		{"PATCH", "/api/settings", "403", "75ms"},
	}

	// Create table with styles matching our app
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(60),
	)

	// Apply our styles
	tbl = ApplyTableStyles(tbl)

	// Test with different cursor positions
	for cursor := 0; cursor < len(rows); cursor++ {
		tbl.SetCursor(cursor)
		output := tbl.View()

		t.Logf("\n========== CURSOR AT POSITION %d (Row: %v) ==========\n", cursor, rows[cursor])

		lines := strings.Split(output, "\n")
		for i, line := range lines {
			// Show line number and content
			t.Logf("Line %2d: %q", i, line)
		}

		t.Logf("Total lines: %d\n", len(lines))
	}
}

func TestColorizeWithRealTable(t *testing.T) {
	// Create the same table
	columns := []table.Column{
		{Title: "Method", Width: 10},
		{Title: "URL", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Duration", Width: 10},
	}

	rows := []table.Row{
		{"GET", "/api/users", "200", "100ms"},
		{"POST", "/api/orders", "201", "250ms"},
		{"DELETE", "/api/items/123", "404", "50ms"},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(60),
	)
	tbl = ApplyTableStyles(tbl)

	// Test cursor at position 1 (POST row)
	tbl.SetCursor(1)
	tableView := tbl.View()

	t.Logf("\n========== BEFORE COLORIZATION (cursor=1) ==========")
	t.Logf("%s", tableView)

	// Apply colorization
	colorized := ColorizeHARTableOutput(tableView, 1, rows)

	t.Logf("\n========== AFTER COLORIZATION (cursor=1) ==========")
	t.Logf("%s", colorized)

	// Check that the selected line (POST) is NOT colorized
	lines := strings.Split(colorized, "\n")
	for i, line := range lines {
		if strings.Contains(line, "POST") {
			t.Logf("\nLine %d contains POST: %q", i, line)
			// Should NOT contain ANSI color codes for POST since it's selected
			if strings.Contains(line, "\033[") && strings.Contains(line, "POST") {
				t.Logf("  ^ Contains ANSI codes (might be wrong if this is the selected line)")
			} else {
				t.Logf("  ^ No ANSI codes (correct for selected line)")
			}
		}
	}
}
