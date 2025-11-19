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

type indexProgressMsg struct {
	bytesRead    int64
	totalBytes   int64
	entriesSoFar int
}

func (m *HARViewModel) listenForProgress() tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-m.progressChan
		if !ok {
			return nil // channel closed
		}
		return indexProgressMsg{
			bytesRead:    progress.BytesRead,
			totalBytes:   progress.TotalBytes,
			entriesSoFar: progress.EntriesSoFar,
		}
	}
}

func (m *HARViewModel) startIndexing() tea.Cmd {
	indexCmd := func() tea.Msg {
		start := time.Now()

		streamer, err := motor.NewHARStreamer(m.fileName, motor.DefaultStreamerOptions())
		if err != nil {
			return indexErrorMsg{err: err}
		}

		// motor closes the channel when done
		if err := streamer.InitializeWithProgress(context.Background(), m.progressChan); err != nil {
			return indexErrorMsg{err: err}
		}

		index := streamer.GetIndex()

		return indexCompleteMsg{
			index:    index,
			streamer: streamer,
			duration: time.Since(start),
		}
	}

	return tea.Batch(indexCmd, m.listenForProgress())
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
	content.WriteString("\n\n")

	// show progress bar if we have progress data
	if m.indexingPercent > 0 {
		content.WriteString(m.progressBar.ViewAs(m.indexingPercent))
		content.WriteString("  ")
		content.WriteString(messageStyle.Render(fmt.Sprintf("Processed %d entries (%.1f%%)",
			m.indexingEntries, m.indexingPercent*100)))
	} else if m.indexingMessage != "" {
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