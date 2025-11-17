package tui

import (
    "github.com/charmbracelet/bubbles/v2/table"
    "github.com/charmbracelet/lipgloss/v2"
)

// matching vacuum's color scheme
var (
    ColorPink       = "#FF10F0"
    ColorBlue       = "#00BFFF"
    ColorGrey       = "#808080"
    ColorDarkGrey   = "#404040"
    ColorSubtlePink = "#3A1A38"

    ColorGreen  = "#10FF10"
    ColorYellow = "#FFD700"
    ColorRed    = "#FF3030"

    ColorWhite = "#FFFFFF"
    ColorBlack = "#000000"
)

var (
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color(ColorPink))

    SubtitleStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGrey))

    HeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color(ColorBlue))

    SelectedStyle = lipgloss.NewStyle().
        Background(lipgloss.Color(ColorSubtlePink)).
        Foreground(lipgloss.Color(ColorPink))

    StatusOKStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGreen))

    StatusWarningStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorYellow))

    StatusErrorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorRed))

    BorderStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color(ColorBlue))

    ViewportTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color(ColorBlue)).
        Background(lipgloss.Color(ColorDarkGrey)).
        Padding(0, 1)

    HelpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorGrey))

    HelpKeyStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorPink))

    ErrorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color(ColorRed)).
        Bold(true)
)

// currently using default white styles; pb33f theme will be activated later
func ApplyTableStyles(t table.Model) table.Model {
    s := table.DefaultStyles()
    t.SetStyles(s)
    return t
}

func GetStatusColor(statusCode int) string {
    switch {
    case statusCode >= 200 && statusCode < 300:
        return ColorGreen
    case statusCode >= 300 && statusCode < 400:
        return ColorBlue
    case statusCode >= 400 && statusCode < 500:
        return ColorYellow
    case statusCode >= 500:
        return ColorRed
    default:
        return ColorGrey
    }
}

// GetMethodColor returns the appropriate color for HTTP method
func GetMethodColor(method string) string {
    switch method {
    case "GET":
        return ColorGreen
    case "POST":
        return ColorBlue
    case "PUT", "PATCH":
        return ColorYellow
    case "DELETE":
        return ColorRed
    default:
        return ColorGrey
    }
}
