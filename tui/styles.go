package tui

import (
    "github.com/charmbracelet/bubbles/v2/table"
    "github.com/charmbracelet/lipgloss/v2"
)

// Color constants matching vacuum EXACTLY
var (
    RGBBlue       = lipgloss.Color("45")
    RGBPink       = lipgloss.Color("201")
    RGBRed        = lipgloss.Color("196")
    RGBYellow     = lipgloss.Color("220")
    RGBGreen      = lipgloss.Color("46")
    RGBGrey       = lipgloss.Color("246")
    RGBSubtlePink = lipgloss.Color("#2a1a2a")
)

// General styles
var (
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(RGBPink)

    SubtitleStyle = lipgloss.NewStyle().
        Foreground(RGBGrey)

    HeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(RGBBlue)

    SelectedStyle = lipgloss.NewStyle().
        Background(RGBSubtlePink).
        Foreground(RGBPink)

    StatusOKStyle = lipgloss.NewStyle().
        Foreground(RGBGreen)

    StatusWarningStyle = lipgloss.NewStyle().
        Foreground(RGBYellow)

    StatusErrorStyle = lipgloss.NewStyle().
        Foreground(RGBRed)

    BorderStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(RGBBlue)

    ViewportTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(RGBBlue).
        Background(RGBGrey).
        Padding(0, 1)

    HelpStyle = lipgloss.NewStyle().
        Foreground(RGBGrey)

    HelpKeyStyle = lipgloss.NewStyle().
        Foreground(RGBPink)

    ErrorStyle = lipgloss.NewStyle().
        Foreground(RGBRed).
        Bold(true)
)

// Table colorization styles for methods and status codes
var (
    // HTTP Methods
    StyleMethodGreen  = lipgloss.NewStyle().Foreground(RGBGreen)  // GET, QUERY
    StyleMethodYellow = lipgloss.NewStyle().Foreground(RGBYellow) // PATCH
    StyleMethodBlue   = lipgloss.NewStyle().Foreground(RGBBlue)   // PUT, POST
    StyleMethodRed    = lipgloss.NewStyle().Foreground(RGBRed)    // DELETE

    // Status codes
    StyleStatus4xx = lipgloss.NewStyle().Foreground(RGBYellow) // 4xx errors
    StyleStatus5xx = lipgloss.NewStyle().Foreground(RGBRed)    // 5xx errors

    // Duration (faint like entry count)
    StyleDurationFaint = lipgloss.NewStyle().Faint(true)
)

// ApplyTableStyles applies the Vacuum table theme to match exactly
func ApplyTableStyles(t table.Model) table.Model {
    s := table.DefaultStyles()

    s.Header = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(RGBPink).
        BorderBottom(true).
        BorderLeft(false).
        BorderRight(false).
        BorderTop(false).
        Foreground(RGBPink).
        Bold(true).
        Padding(0, 1)

    s.Selected = lipgloss.NewStyle().
        Bold(true).
        Foreground(RGBPink).
        Background(RGBSubtlePink).
        Padding(0, 0)

    s.Cell = lipgloss.NewStyle().
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(RGBPink).
        BorderRight(false).
        Padding(0, 1)

    t.SetStyles(s)
    return t
}
