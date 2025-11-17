package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/v2/table"
)

// ColorizeHARTableOutput adds color codes to the rendered table output
// following the vacuum pattern of post-processing the table.View() output
func ColorizeHARTableOutput(tableView string, cursor int, rows []table.Row) string {
    lines := strings.Split(tableView, "\n")

    // Table structure: line 0 = header, line 1 = border, line 2+ = data rows
    // cursor 0 maps to line 2, cursor 1 to line 3, etc.
    selectedLineIdx := cursor + 2

    var result strings.Builder
    for i, line := range lines {
        // Only colorize non-selected rows (matching Vacuum pattern)
        // Selected rows are already styled by the table with pink background
        if i >= 1 && i != selectedLineIdx {
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
