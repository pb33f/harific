package tui

import (
    "regexp"
    "strconv"
    "strings"

    "github.com/charmbracelet/bubbles/v2/table"
)

// pre-compiled regex for status code matching (compiled once at startup)
var statusCodeRegex = regexp.MustCompile(`\s(\d{3})\s`)

// pre-rendered method strings (populated in init() after styles are ready)
var (
    renderedGET    string
    renderedQUERY  string
    renderedPATCH  string
    renderedPUT    string
    renderedPOST   string
    renderedDELETE string
)

func init() {
    // initialize rendered method strings after package variables are ready
    renderedGET = StyleMethodGreen.Render("GET")
    renderedQUERY = StyleMethodGreen.Render("QUERY")
    renderedPATCH = StyleMethodYellow.Render("PATCH")
    renderedPUT = StyleMethodBlue.Render("PUT")
    renderedPOST = StyleMethodBlue.Render("POST")
    renderedDELETE = StyleMethodRed.Render("DELETE")
}

// ColorizeHARTableOutput adds color codes to the rendered table output
// following the vacuum pattern of post-processing the table.View() output
func ColorizeHARTableOutput(tableView string, cursor int, rows []table.Row) string {
    lines := strings.Split(tableView, "\n")

    // Build unique identifier from selected row (method + URL + status + duration)
    // This handles cases where the table doesn't apply background styling when scrolled
    var selectedIdentifier string
    if cursor >= 0 && cursor < len(rows) && len(rows[cursor]) >= 4 {
        // Combine all columns to create unique identifier that minimizes false matches
        selectedIdentifier = rows[cursor][0] + rows[cursor][1] + rows[cursor][2] + rows[cursor][3]
    }

    // Also check for pink background ANSI code (works when selection is at top/bottom of viewport)
    selectedLineMarker := "\x1b[1;38;5;201;48;2;42;26;42m"

    var result strings.Builder
    result.Grow(len(tableView)) // pre-allocate to avoid reallocations during string building

    for i, line := range lines {
        // Check if this line is selected (either has background marker OR matches all column data)
        isSelectedLine := strings.Contains(line, selectedLineMarker) ||
            (selectedIdentifier != "" && strings.Contains(line, selectedIdentifier))

        // Only colorize non-selected rows (matching Vacuum pattern)
        // Selected rows are already styled by the table with pink background
        if i >= 1 && !isSelectedLine {
            // Colorize HTTP methods
            line = colorizeHTTPMethods(line)

            // Colorize status codes
            line = colorizeStatusCodes(line)

            // Colorize durations
            line = colorizeDurations(line)
        }

        result.WriteString(line)
        if i < len(lines)-1 {
            result.WriteString("\n")
        }
    }

    return result.String()
}

// colorizeHTTPMethods applies colors to HTTP method names using pre-rendered strings
func colorizeHTTPMethods(line string) string {
    // use pre-rendered strings to avoid calling Render() on every line
    line = strings.ReplaceAll(line, " GET ", " "+renderedGET+" ")
    line = strings.ReplaceAll(line, " QUERY ", " "+renderedQUERY+" ")
    line = strings.ReplaceAll(line, " PATCH ", " "+renderedPATCH+" ")
    line = strings.ReplaceAll(line, " PUT ", " "+renderedPUT+" ")
    line = strings.ReplaceAll(line, " POST ", " "+renderedPOST+" ")
    line = strings.ReplaceAll(line, " DELETE ", " "+renderedDELETE+" ")
    // HEAD, OPTIONS, TRACE remain default (no colorization)
    return line
}

// colorizeStatusCodes applies colors to status codes using regex
func colorizeStatusCodes(line string) string {
    // use pre-compiled regex to find 3-digit status code
    matches := statusCodeRegex.FindStringSubmatch(line)
    if len(matches) < 2 {
        return line // no status code found
    }

    statusStr := matches[1] // "404", "500", etc.
    statusCode, err := strconv.Atoi(statusStr)
    if err != nil {
        return line
    }

    // colorize based on range
    if statusCode >= 400 && statusCode < 500 {
        // 4xx - yellow
        colored := " " + StyleStatus4xx.Render(statusStr) + " "
        line = strings.Replace(line, " "+statusStr+" ", colored, 1)
    } else if statusCode >= 500 && statusCode < 600 {
        // 5xx - red
        colored := " " + StyleStatus5xx.Render(statusStr) + " "
        line = strings.Replace(line, " "+statusStr+" ", colored, 1)
    }

    return line
}

// colorizeDurations applies faint style to duration values
func colorizeDurations(line string) string {
    // Durations end with "ms" or "s" and appear in the last column
    // We need to find patterns like "123ms", "1.23s", "571ms", etc.
    // Must be careful not to match URLs or paths that end with "s"

    // Find duration patterns - common formats: "12.34s", "123ms", "1234ms", "571ms"
    if strings.Contains(line, "ms") || strings.Contains(line, "s") {
        // Split by spaces to find duration tokens
        parts := strings.Split(line, " ")
        for i, part := range parts {
            if isDuration(part) {
                parts[i] = StyleDurationFaint.Render(part)
            }
        }
        line = strings.Join(parts, " ")
    }

    return line
}

// isDuration checks if a string is a valid duration (not a path or other text)
func isDuration(s string) bool {
    if s == "" {
        return false
    }

    // Must start with a digit
    if s[0] < '0' || s[0] > '9' {
        return false
    }

    // Check for valid time unit suffixes: ms, s, m, h
    var valueStr string
    if strings.HasSuffix(s, "ms") {
        valueStr = strings.TrimSuffix(s, "ms")
    } else if strings.HasSuffix(s, "s") {
        valueStr = strings.TrimSuffix(s, "s")
    } else if strings.HasSuffix(s, "m") {
        valueStr = strings.TrimSuffix(s, "m")
    } else if strings.HasSuffix(s, "h") {
        valueStr = strings.TrimSuffix(s, "h")
    } else {
        return false // No valid time unit suffix
    }

    // Value must not be empty
    if len(valueStr) == 0 {
        return false
    }

    // Value should only contain digits and at most one decimal point
    // This excludes paths like "/api/users" or identifiers like "5u7hmsls"
    dotCount := 0
    for _, c := range valueStr {
        if c == '.' {
            dotCount++
            if dotCount > 1 {
                return false // Multiple dots not valid
            }
        } else if c < '0' || c > '9' {
            return false // Non-digit character found (excludes paths and identifiers)
        }
    }

    return true
}

