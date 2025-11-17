package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/v2/table"
)

// TestActualScrollingBehavior simulates real scrolling by moving cursor one step at a time
func TestActualScrollingBehavior(t *testing.T) {
	columns := []table.Column{
		{Title: "Method", Width: 10},
		{Title: "URL", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Duration", Width: 10},
	}

	// Create enough rows to require scrolling
	rows := make([]table.Row, 30)
	for i := 0; i < 30; i++ {
		method := "GET"
		status := "200"
		if i%3 == 0 {
			method = "POST"
			status = "201"
		} else if i%3 == 1 {
			method = "DELETE"
			status = "404"
		}
		url := fmt.Sprintf("/api/endpoint/row%03d", i)
		rows[i] = table.Row{method, url, status, "100ms"}
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10), // Show ~8 data rows
		table.WithWidth(60),
	)
	tbl = ApplyTableStyles(tbl)

	// Simulate scrolling by moving cursor incrementally
	for cursor := 0; cursor < 20; cursor++ {
		tbl.SetCursor(cursor)
		output := tbl.View()

		// Apply colorization
		colorized := ColorizeHARTableOutput(output, cursor, rows)

		// Verify selected row is NOT colorized
		selectedMethod := rows[cursor][0]
		selectedURL := rows[cursor][1]

		lines := strings.Split(colorized, "\n")
		foundSelected := false

		for _, line := range lines {
			if strings.Contains(line, selectedURL) {
				foundSelected = true

				// Check if this line was incorrectly colorized
				if selectedMethod == "POST" && strings.Contains(line, "\x1b[38;5;45m") {
					t.Errorf("CURSOR %d: Selected POST was colorized (should not be)", cursor)
					t.Logf("Line: %q", line)
				} else if selectedMethod == "DELETE" && strings.Contains(line, "\x1b[38;5;196m") {
					t.Errorf("CURSOR %d: Selected DELETE was colorized (should not be)", cursor)
					t.Logf("Line: %q", line)
				} else if selectedMethod == "GET" && strings.Contains(line, "\x1b[38;5;46m") {
					t.Errorf("CURSOR %d: Selected GET was colorized (should not be)", cursor)
					t.Logf("Line: %q", line)
				}
			}
		}

		if !foundSelected {
			t.Logf("CURSOR %d: Selected row not visible in viewport (this is OK)", cursor)
		}
	}
}
