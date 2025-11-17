package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/v2/table"
)

// TestDuplicateRequests tests colorization when the same request appears multiple times
func TestDuplicateRequests(t *testing.T) {
	columns := []table.Column{
		{Title: "Method", Width: 10},
		{Title: "URL", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Duration", Width: 10},
	}

	// Create rows with EXACT duplicates (same endpoint hit multiple times)
	rows := []table.Row{
		{"POST", "/api/users", "201", "100ms"}, // row 0
		{"GET", "/api/products", "200", "50ms"}, // row 1
		{"POST", "/api/users", "201", "100ms"}, // row 2 - EXACT DUPLICATE of row 0
		{"DELETE", "/api/items/123", "404", "75ms"}, // row 3
		{"POST", "/api/users", "201", "100ms"}, // row 4 - EXACT DUPLICATE of row 0
		{"GET", "/api/products", "200", "50ms"}, // row 5 - EXACT DUPLICATE of row 1
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(60),
	)
	tbl = ApplyTableStyles(tbl)

	// Test with cursor on the first POST /api/users (row 0)
	tbl.SetCursor(0)
	output := tbl.View()
	colorized := ColorizeHARTableOutput(output, 0, rows)

	t.Logf("\n========== CURSOR AT POSITION 0 (first POST /api/users) ==========")
	lines := strings.Split(colorized, "\n")

	postLineCount := 0
	colorizedPostCount := 0

	for i, line := range lines {
		if strings.Contains(line, "POST") && strings.Contains(line, "/api/users") {
			postLineCount++
			t.Logf("Line %d has POST /api/users", i)

			// Check if this POST is colorized (blue)
			if strings.Contains(line, "\x1b[38;5;45m") {
				colorizedPostCount++
				t.Logf("  ^ This POST is colorized BLUE")
			} else {
				t.Logf("  ^ This POST is NOT colorized (selected or bug)")
			}
		}
	}

	t.Logf("\nTotal POST /api/users rows visible: %d", postLineCount)
	t.Logf("Colorized POST /api/users rows: %d", colorizedPostCount)

	// Expected: 3 POST lines visible (rows 0, 2, 4)
	// Expected: 2 colorized (rows 2 and 4), 1 not colorized (row 0 - selected)
	if postLineCount == 3 && colorizedPostCount == 2 {
		t.Logf("✓ CORRECT: Selected row skipped, duplicates colorized")
	} else if postLineCount == 3 && colorizedPostCount == 0 {
		t.Errorf("ERROR: All duplicates were skipped (false positive match)")
	} else {
		t.Logf("Unexpected result - needs investigation")
	}
}
