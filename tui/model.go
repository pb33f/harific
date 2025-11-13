package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	"github.com/charmbracelet/bubbles/v2/viewport"
	"github.com/pb33f/braid/motor"
	"github.com/pb33f/harhar"
)

// ViewMode represents the different view states
type ViewMode int

const (
	ViewModeTable ViewMode = iota
	ViewModeTableWithSplit
)

type HARViewModel struct {
	table           table.Model
	allEntries      []*motor.EntryMetadata
	rows            []table.Row
	columns         []table.Column

	streamer        motor.HARStreamer
	index           *motor.Index
	selectedEntry   *harhar.Entry
	selectedIndex   int

	viewMode        ViewMode
	width           int
	height          int
	ready           bool
	quitting        bool

	requestViewport  viewport.Model
	responseViewport viewport.Model
	splitVisible     bool

	fileName        string

	loadState       LoadState
	loadingSpinner  spinner.Model
	indexingMessage string
	indexingTime    time.Duration

	err             error
}

func NewHARViewModel(fileName string) (*HARViewModel, error) {
	columns := []table.Column{
		{Title: "Method", Width: methodColumnWidth},
		{Title: "URL", Width: maxURLDisplayLength},
		{Title: "Status", Width: statusColumnWidth},
		{Title: "Duration", Width: durationColumnWidth},
	}

	m := &HARViewModel{
		fileName:        fileName,
		columns:         columns,
		viewMode:        ViewModeTable,
		selectedIndex:   0,
		loadState:       LoadStateLoading,
		loadingSpinner:  createLoadingSpinner(),
		indexingMessage: "Building index...",
	}

	return m, nil
}

func (m *HARViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadingSpinner.Tick,
		m.startIndexing(),
	)
}

func (m *HARViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.loadState == LoadStateLoading {
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case indexCompleteMsg:
		m.loadState = LoadStateLoaded
		m.index = msg.index
		m.streamer = msg.streamer
		m.allEntries = msg.index.Entries
		m.indexingTime = msg.duration

		if m.width > 0 && m.height > 0 {
			m.initializeTable()
			m.ready = true
		}
		return m, nil

	case indexErrorMsg:
		m.loadState = LoadStateError
		m.err = msg.err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.loadState == LoadStateLoaded && !m.ready && m.index != nil {
			m.initializeTable()
			m.ready = true
		} else if m.ready {
			m.updateTableDimensions()
		}

		if m.splitVisible {
			m.updateViewportDimensions()
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter", "return":
			if m.loadState == LoadStateLoaded {
				m.toggleSplitView()
				if m.splitVisible {
					if err := m.loadSelectedEntry(); err != nil {
						m.err = err
					}
				}
			}
			return m, nil

		case "esc":
			if m.loadState == LoadStateLoaded && m.splitVisible {
				m.splitVisible = false
				m.viewMode = ViewModeTable
				m.updateTableDimensions()
			}
			return m, nil
		}
	}

	if m.loadState == LoadStateLoaded {
		if !m.splitVisible {
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)

			if m.table.Cursor() != m.selectedIndex {
				m.selectedIndex = m.table.Cursor()
			}
		}

		if m.splitVisible {
			m.requestViewport, cmd = m.requestViewport.Update(msg)
			cmds = append(cmds, cmd)

			m.responseViewport, cmd = m.responseViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *HARViewModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.loadState {
	case LoadStateLoading:
		return m.renderLoadingView()
	case LoadStateError:
		return m.renderErrorView()
	case LoadStateLoaded:
		if !m.ready {
			return "Initializing..."
		}
		return m.render()
	default:
		return "Unknown state"
	}
}

func (m *HARViewModel) initializeTable() {
	m.buildTableRows()

	tableHeight := m.height - tableVerticalPadding
	if m.splitVisible {
		tableHeight = (m.height - tableVerticalPadding) / 2
	}

	m.table = table.New(
		table.WithColumns(m.columns),
		table.WithRows(m.rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithWidth(m.width),
	)

	m.table = ApplyTableStyles(m.table)
}

func (m *HARViewModel) updateTableDimensions() {
	tableHeight := m.height - tableVerticalPadding
	if m.splitVisible {
		tableHeight = (m.height - tableVerticalPadding) / 2
	}
	m.table.SetHeight(tableHeight)
	m.table.SetWidth(m.width)

	m.adjustColumnWidths()
}

func (m *HARViewModel) updateViewportDimensions() {
	splitHeight := (m.height - tableVerticalPadding) / 2 - splitPanelPadding
	splitWidth := (m.width / 2) - splitPanelPadding

	if m.requestViewport.Width() == 0 {
		m.requestViewport = viewport.New(viewport.WithWidth(splitWidth), viewport.WithHeight(splitHeight))
		m.responseViewport = viewport.New(viewport.WithWidth(splitWidth), viewport.WithHeight(splitHeight))
	} else {
		m.requestViewport.SetWidth(splitWidth)
		m.requestViewport.SetHeight(splitHeight)
		m.responseViewport.SetWidth(splitWidth)
		m.responseViewport.SetHeight(splitHeight)
	}
}

func (m *HARViewModel) toggleSplitView() {
	if m.viewMode == ViewModeTable {
		m.viewMode = ViewModeTableWithSplit
		m.splitVisible = true
		m.updateTableDimensions()
		m.updateViewportDimensions()
	} else {
		m.viewMode = ViewModeTable
		m.splitVisible = false
		m.updateTableDimensions()
	}
}

func (m *HARViewModel) loadSelectedEntry() error {
	if m.selectedIndex >= len(m.allEntries) {
		return nil
	}

	ctx := context.Background()
	entry, err := m.streamer.GetEntry(ctx, m.selectedIndex)
	if err != nil {
		return err
	}

	m.selectedEntry = entry

	if entry != nil {
		m.updateViewportContent()
	}

	return nil
}

func (m *HARViewModel) adjustColumnWidths() {
	urlWidth := m.width - methodColumnWidth - statusColumnWidth - durationColumnWidth - borderPadding
	if urlWidth < minURLColumnWidth {
		urlWidth = minURLColumnWidth
	}

	m.columns[0].Width = methodColumnWidth
	m.columns[1].Width = urlWidth
	m.columns[2].Width = statusColumnWidth
	m.columns[3].Width = durationColumnWidth

	m.table.SetColumns(m.columns)
}