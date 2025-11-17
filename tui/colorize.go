package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/v2/table"
)

// ColorizeHARTableOutput adds color codes to the rendered table output
// following the vacuum pattern of post-processing the table.View() output
func ColorizeHARTableOutput(tableView string, cursor int, rows []table.Row) string {
    lines := strings.Split(tableView, "\n")

    var result strings.Builder
    for i, line := range lines {
        // Skip header row (i == 0)
        if i >= 1 {
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

// colorizeHTTPMethods applies colors to HTTP method names
func colorizeHTTPMethods(line string) string {
    // GET and QUERY - green
    line = strings.ReplaceAll(line, " GET ", " "+StyleMethodGreen.Render("GET")+" ")
    line = strings.ReplaceAll(line, " QUERY ", " "+StyleMethodGreen.Render("QUERY")+" ")

    // PATCH - yellow
    line = strings.ReplaceAll(line, " PATCH ", " "+StyleMethodYellow.Render("PATCH")+" ")

    // PUT and POST - blue
    line = strings.ReplaceAll(line, " PUT ", " "+StyleMethodBlue.Render("PUT")+" ")
    line = strings.ReplaceAll(line, " POST ", " "+StyleMethodBlue.Render("POST")+" ")

    // DELETE - red
    line = strings.ReplaceAll(line, " DELETE ", " "+StyleMethodRed.Render("DELETE")+" ")

    // HEAD, OPTIONS, TRACE remain default (no colorization)

    return line
}

// colorizeStatusCodes applies colors to status codes
func colorizeStatusCodes(line string) string {
    // Check for 4xx status codes (yellow)
    for status := 400; status < 500; status++ {
        statusStr := " " + intToString(status) + " "
        if strings.Contains(line, statusStr) {
            line = strings.Replace(line, statusStr, " "+StyleStatus4xx.Render(intToString(status))+" ", 1)
            break
        }
    }

    // Check for 5xx status codes (red)
    for status := 500; status < 600; status++ {
        statusStr := " " + intToString(status) + " "
        if strings.Contains(line, statusStr) {
            line = strings.Replace(line, statusStr, " "+StyleStatus5xx.Render(intToString(status))+" ", 1)
            break
        }
    }

    return line
}

// colorizeDurations applies faint style to duration values
func colorizeDurations(line string) string {
    // Durations end with "ms" or "s" and appear in the last column
    // We need to find patterns like "123ms", "1.23s", "571ms", etc.

    // Find duration patterns - common formats: "12.34s", "123ms", "1234ms", "571ms"
    if strings.Contains(line, "ms ") || strings.Contains(line, "s ") {
        // Split by spaces to find duration tokens
        parts := strings.Split(line, " ")
        for i, part := range parts {
            if strings.HasSuffix(part, "ms") || strings.HasSuffix(part, "s") {
                // Check if it's actually a duration (contains digits)
                if containsDigit(part) {
                    parts[i] = StyleDurationFaint.Render(part)
                }
            }
        }
        line = strings.Join(parts, " ")
    }

    return line
}

// intToString converts an int to string without importing strconv
func intToString(n int) string {
    if n == 0 {
        return "0"
    }

    // Build string representation
    digits := []byte{}
    for n > 0 {
        digits = append([]byte{byte('0' + n%10)}, digits...)
        n /= 10
    }
    return string(digits)
}

// containsDigit checks if a string contains at least one digit
func containsDigit(s string) bool {
    for _, c := range s {
        if c >= '0' && c <= '9' {
            return true
        }
    }
    return false
}
