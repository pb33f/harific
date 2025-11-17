package tui

import (
	"context"
	"fmt"
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
	spinnerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(RGBPink)

	fileInfoStyle := lipgloss.NewStyle().
		Foreground(RGBGrey)

	title := titleStyle.Render("Loading HAR File")
	fileInfo := fileInfoStyle.Render(fmt.Sprintf("\n%s", m.fileName))

	spinnerText := fmt.Sprintf("%s %s%s", m.loadingSpinner.View(), title, fileInfo)

	if m.indexingMessage != "" {
		messageStyle := lipgloss.NewStyle().
			Foreground(RGBBlue).
			MarginTop(2)
		spinnerText += "\n\n" + messageStyle.Render(m.indexingMessage)
	}

	return spinnerStyle.Render(spinnerText)
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