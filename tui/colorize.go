package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/v2/table"
)


// pre-rendered method strings to avoid repeated style.Render() calls in hot path
var (
    renderedGET    string
    renderedQUERY  string
    renderedPATCH  string
    renderedPUT    string
    renderedPOST   string
    renderedDELETE string
)

func init() {
    renderedGET = StyleMethodGreen.Render("GET")
    renderedQUERY = StyleMethodGreen.Render("QUERY")
    renderedPATCH = StyleMethodYellow.Render("PATCH")
    renderedPUT = StyleMethodBlue.Render("PUT")
    renderedPOST = StyleMethodBlue.Render("POST")
    renderedDELETE = StyleMethodRed.Render("DELETE")
}

// colorizes table output following vacuum pattern - skips selected row to preserve background
func ColorizeHARTableOutput(tableView string, cursor int, rows []table.Row) string {
    lines := strings.Split(tableView, "\n")

    // build unique identifier from selected row to handle cases where table background fails when scrolled
    var selectedIdentifier string
    if cursor >= 0 && cursor < len(rows) && len(rows[cursor]) >= 4 {
        selectedIdentifier = rows[cursor][0] + rows[cursor][1] + rows[cursor][2] + rows[cursor][3]
    }

    // ANSI escape sequence for pink background (matches table selected style from styles.go)
    selectedLineMarker := "\x1b[1;38;5;201;48;2;42;26;42m"

    var result strings.Builder
    // estimate output size: input + ANSI overhead per line (~40 bytes per colorized line)
    estimatedSize := len(tableView) + (len(lines) * 40)
    result.Grow(estimatedSize)

    for i, line := range lines {
        isSelectedLine := strings.Contains(line, selectedLineMarker) ||
            (selectedIdentifier != "" && strings.Contains(line, selectedIdentifier))

        // skip header row (i=0) and selected rows (already styled by table)
        if i >= 1 && !isSelectedLine {
            line = colorizeHTTPMethods(line)
            line = colorizeStatusCodes(line)
            line = colorizeDurations(line)
        }

        result.WriteString(line)
        if i < len(lines)-1 {
            result.WriteString("\n")
        }
    }

    return result.String()
}

// colorizes HTTP method using pre-rendered strings with early-return optimization
func colorizeHTTPMethods(line string) string {
    // ordered by frequency: GET most common, QUERY least common
    if strings.Contains(line, " GET ") {
        return strings.Replace(line, " GET ", " "+renderedGET+" ", 1)
    }
    if strings.Contains(line, " POST ") {
        return strings.Replace(line, " POST ", " "+renderedPOST+" ", 1)
    }
    if strings.Contains(line, " PUT ") {
        return strings.Replace(line, " PUT ", " "+renderedPUT+" ", 1)
    }
    if strings.Contains(line, " DELETE ") {
        return strings.Replace(line, " DELETE ", " "+renderedDELETE+" ", 1)
    }
    if strings.Contains(line, " PATCH ") {
        return strings.Replace(line, " PATCH ", " "+renderedPATCH+" ", 1)
    }
    if strings.Contains(line, " QUERY ") {
        return strings.Replace(line, " QUERY ", " "+renderedQUERY+" ", 1)
    }
    return line
}

// colorizes 4xx (yellow) and 5xx (red) status codes using manual byte scanning
func colorizeStatusCodes(line string) string {
    // find " NNN " pattern (3 digits surrounded by spaces)
    for i := 0; i < len(line)-4; i++ {
        if line[i] == ' ' &&
            line[i+1] >= '0' && line[i+1] <= '9' &&
            line[i+2] >= '0' && line[i+2] <= '9' &&
            line[i+3] >= '0' && line[i+3] <= '9' &&
            line[i+4] == ' ' {

            // parse status code from digits
            statusCode := int(line[i+1]-'0')*100 + int(line[i+2]-'0')*10 + int(line[i+3]-'0')

            if statusCode >= 400 && statusCode < 500 {
                statusStr := line[i+1 : i+4]
                colored := " " + StyleStatus4xx.Render(statusStr) + " "
                return line[:i] + colored + line[i+5:]
            } else if statusCode >= 500 && statusCode < 600 {
                statusStr := line[i+1 : i+4]
                colored := " " + StyleStatus5xx.Render(statusStr) + " "
                return line[:i] + colored + line[i+5:]
            }
            return line
        }
    }
    return line
}

// colorizes duration values in last column with faint style
func colorizeDurations(line string) string {
    lastSpaceIdx := strings.LastIndexByte(line, ' ')
    if lastSpaceIdx == -1 {
        return line
    }

    durationPart := line[lastSpaceIdx+1:]
    if isDuration(durationPart) {
        styledDuration := StyleDurationFaint.Render(durationPart)
        return line[:lastSpaceIdx+1] + styledDuration
    }

    return line
}

// isDuration validates if string is a time duration (e.g., "150ms", "2.5s")
// rejects URLs, paths, and random identifiers by requiring digit-only numeric portion
func isDuration(s string) bool {
    if s == "" {
        return false
    }

    if s[0] < '0' || s[0] > '9' {
        return false
    }

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
        return false
    }

    if len(valueStr) == 0 {
        return false
    }

    // reject paths ("/api/users") and identifiers ("5u7hmsls")
    dotCount := 0
    for _, c := range valueStr {
        if c == '.' {
            dotCount++
            if dotCount > 1 {
                return false
            }
        } else if c < '0' || c > '9' {
            return false
        }
    }

    return true
}

