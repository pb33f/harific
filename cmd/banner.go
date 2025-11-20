package cmd

import (
    "image/color"

    "github.com/charmbracelet/lipgloss/v2"
    "github.com/pb33f/braid/tui"
)

const pb33fASCII = `@@@@@@@   @@@@@@@   @@@@@@   @@@@@@   @@@@@@@@
@@@@@@@@  @@@@@@@@  @@@@@@@  @@@@@@@  @@@@@@@@
@@!  @@@  @@!  @@@      @@@      @@@  @@!
!@!  @!@  !@   @!@      @!@      @!@  !@!
@!@@!@!   @!@!@!@   @!@!!@   @!@!!@   @!!!:!
!!@!!!    !!!@!!!!  !!@!@!   !!@!@!   !!!!!:
!!:       !!:  !!!      !!:      !!:  !!:
:!:       :!:  !:!      :!:      :!:  :!:
 ::        :: ::::  :: ::::  :: ::::   ::
 :        :: : ::    : : :    : : :    :      `

// RenderBanner returns the styled pb33f banner for the help screen
func RenderBanner() string {
    // Create the gradient-like effect using different shades of pink
    bannerStyle := lipgloss.NewStyle().
        Foreground(tui.RGBPink).
        Bold(true)

    // Create a subtitle style
    subtitleStyle := lipgloss.NewStyle().
        Foreground(tui.RGBBlue).
        Italic(true)

    // Center the banner
    containerStyle := lipgloss.NewStyle().
        Align(lipgloss.Center).
        MarginBottom(1)

    banner := bannerStyle.Render(pb33fASCII)
    subtitle := subtitleStyle.Render("pb33f - the home of enterprise OpenAPI tools")

    return containerStyle.Render(banner + "\n" + subtitle)
}

// RenderColorfulBanner returns a more colorful version with alternating colors
func RenderColorfulBanner() string {
    lines := []string{
        "@@@@@@@   @@@@@@@   @@@@@@   @@@@@@   @@@@@@@@",
        "@@@@@@@@  @@@@@@@@  @@@@@@@  @@@@@@@  @@@@@@@@",
        "@@!  @@@  @@!  @@@      @@@      @@@  @@!     ",
        "!@!  @!@  !@   @!@      @!@      @!@  !@!     ",
        "@!@@!@!   @!@!@!@   @!@!!@   @!@!!@   @!!!:!  ",
        "!!@!!!    !!!@!!!!  !!@!@!   !!@!@!   !!!!!:  ",
        "!!:       !!:  !!!      !!:      !!:  !!:     ",
        ":!:       :!:  !:!      :!:      :!:  :!:     ",
        " ::        :: ::::  :: ::::  :: ::::   ::     ",
        " :        :: : ::    : : :    : : :    :      ",
    }

    // Create gradient effect with different colors
    colors := []color.Color{
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
        tui.RGBPink,
    }

    var result string
    for i, line := range lines {
        style := lipgloss.NewStyle().
            Foreground(colors[i]).
            Bold(true)
        result += style.Render(line) + "\n"
    }

    // Add subtitle
    subtitleStyle := lipgloss.NewStyle().
        Foreground(tui.RGBBlue).
        Italic(true)

    subtitle := subtitleStyle.Render("https://pb33f.io/harific/")

    // Center everything
    containerStyle := lipgloss.NewStyle().
        Align(lipgloss.Left).
        MarginBottom(1)

    return containerStyle.Render(result + subtitle)
}

// GetBannerWidth returns the width of the banner for layout calculations
func GetBannerWidth() int {
    return 48 // Width of the ASCII art
}

// GetBannerHeight returns the height of the banner including subtitle
func GetBannerHeight() int {
    return 12 // 10 lines of ASCII + 1 subtitle + 1 margin
}
