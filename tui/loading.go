package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/pb33f/braid/motor"
)

type LoadState int

const (
	LoadStateLoading LoadState = iota
	LoadStateLoaded
	LoadStateError
)

type indexCompleteMsg struct {
	index    *motor.Index
	streamer motor.HARStreamer
	duration time.Duration
}

type indexErrorMsg struct {
	err error
}

func (m *HARViewModel) startIndexing() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		streamer, err := motor.NewHARStreamer(m.fileName, motor.DefaultStreamerOptions())
		if err != nil {
			return indexErrorMsg{err: err}
		}

		if err := streamer.Initialize(context.Background()); err != nil {
			return indexErrorMsg{err: err}
		}

		index := streamer.GetIndex()

		return indexCompleteMsg{
			index:    index,
			streamer: streamer,
			duration: time.Since(start),
		}
	}
}

func (m *HARViewModel) renderLoadingView() string {
	// border around the whole screen
	borderStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBBlue)

	// centered content container
	contentStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		Height(m.height - 4).
		Align(lipgloss.Center, lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBPink)

	fileInfoStyle := lipgloss.NewStyle().
		Foreground(RGBGrey)

	messageStyle := lipgloss.NewStyle().
		Foreground(RGBBlue)

	// build centered content
	var content strings.Builder
	content.WriteString(m.loadingSpinner.View())
	content.WriteString(" ")
	content.WriteString(titleStyle.Render("Loading HAR File"))
	content.WriteString("\n")
	content.WriteString(fileInfoStyle.Render(m.fileName))

	if m.indexingMessage != "" {
		content.WriteString("\n\n")
		content.WriteString(messageStyle.Render(m.indexingMessage))
	}

	centeredContent := contentStyle.Render(content.String())
	return borderStyle.Render(centeredContent)
}

func (m *HARViewModel) renderErrorView() string {
	errorStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(RGBRed).
		Bold(true)

	errorMsg := fmt.Sprintf("❌ Error loading HAR file\n\n%v\n\nPress 'q' to quit", m.err)
	return errorStyle.Render(errorMsg)
}

// matching vacuum's Dot spinner
func createLoadingSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(RGBPink)
	return s
}