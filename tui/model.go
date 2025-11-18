package tui

import (
    "context"
    "time"

    "github.com/charmbracelet/bubbles/v2/spinner"
    "github.com/charmbracelet/bubbles/v2/table"
    "github.com/charmbracelet/bubbles/v2/textinput"
    "github.com/charmbracelet/bubbles/v2/viewport"
    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/pb33f/braid/motor"
    "github.com/pb33f/harhar"
)

// ViewMode represents the different view states
type ViewMode int

const (
    ViewModeTable ViewMode = iota
    ViewModeTableWithSplit
    ViewModeTableWithSearch
)

// ViewportFocus represents which viewport has focus
type ViewportFocus int

const (
    ViewportFocusRequest ViewportFocus = iota
    ViewportFocusResponse
)

type HARViewModel struct {
    table      table.Model
    allEntries []*motor.EntryMetadata
    rows       []table.Row
    columns    []table.Column

    streamer      motor.HARStreamer
    index         *motor.Index
    selectedEntry *harhar.Entry
    selectedIndex int

    viewMode ViewMode
    width    int
    height   int
    ready    bool
    quitting bool

    requestViewport  viewport.Model
    responseViewport viewport.Model
    focusedViewport  ViewportFocus

    searchInput   textinput.Model
    searchQuery   string
    searchOptions [3]bool // checkbox states for search locations
    searchCursor  int     // focus position: 0=input, 1-3=checkboxes

    // cache for colorized table during search mode
    cachedColorizedTable string
    cachedTableCursor    int

    fileName string

    loadState       LoadState
    loadingSpinner  spinner.Model
    indexingMessage string
    indexingTime    time.Duration

    err error
}

func NewHARViewModel(fileName string) (*HARViewModel, error) {
    columns := []table.Column{
        {Title: "Method", Width: methodColumnWidth},
        {Title: "URL", Width: maxURLDisplayLength},
        {Title: "Status", Width: statusColumnWidth},
        {Title: "Duration", Width: durationColumnWidth},
    }

    searchInput := textinput.New()
    searchInput.CharLimit = 200

    m := &HARViewModel{
        fileName:        fileName,
        columns:         columns,
        viewMode:        ViewModeTable,
        selectedIndex:   0,
        focusedViewport: ViewportFocusRequest,
        loadState:       LoadStateLoading,
        loadingSpinner:  createLoadingSpinner(),
        indexingMessage: "Building index...",
        searchInput:     searchInput,
        searchCursor:    searchCursorInput,
    }

    return m, nil
}

func (m *HARViewModel) toggleCheckbox() {
    if m.searchCursor > searchCursorInput && m.searchCursor < searchCursorCount {
        m.searchOptions[m.searchCursor-1] = !m.searchOptions[m.searchCursor-1]
    }
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

        if m.viewMode == ViewModeTableWithSplit {
            m.updateViewportDimensions()
        }

    case tea.KeyPressMsg:
        switch msg.String() {
        case "ctrl+c":
            m.quitting = true
            return m, tea.Quit

        case "q":
            // only quit from table view, not from search or split views
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTable {
                m.quitting = true
                return m, tea.Quit
            }
            // don't return - let Q fall through to input handling in search mode

        case "s":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTable {
                cmd := m.toggleSearchView()
                return m, cmd
            }
            return m, nil

        case "enter", "return":
            if m.loadState == LoadStateLoaded {
                if m.viewMode == ViewModeTable {
                    // In table mode, Enter opens split view
                    m.toggleSplitView()
                    if m.viewMode == ViewModeTableWithSplit {
                        if err := m.loadSelectedEntry(); err != nil {
                            m.err = err
                        }
                    }
                } else if m.viewMode == ViewModeTableWithSearch {
                    if m.searchCursor == searchCursorInput {
                        // In search mode on input, Enter triggers search
                        // TODO: implement search execution
                    } else {
                        // On checkbox, Enter toggles like Space
                        m.toggleCheckbox()
                    }
                }
            }
            return m, nil

        case "esc":
            if m.loadState == LoadStateLoaded && m.viewMode != ViewModeTable {
                cmd := m.toggleSearchView()
                return m, cmd
            }
            return m, nil

        case "tab":
            if m.loadState == LoadStateLoaded {
                if m.viewMode == ViewModeTableWithSplit {
                    m.toggleViewportFocus()
                } else if m.viewMode == ViewModeTableWithSearch {
                    m.searchCursor = (m.searchCursor + 1) % searchCursorCount
                    if m.searchCursor == searchCursorInput {
                        return m, m.searchInput.Focus()
                    } else {
                        m.searchInput.Blur()
                    }
                }
            }
            return m, nil

        case "up":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTableWithSearch {
                m.searchCursor--
                if m.searchCursor < 0 {
                    m.searchCursor = searchCursorCount - 1
                }
                if m.searchCursor == searchCursorInput {
                    return m, m.searchInput.Focus()
                } else {
                    m.searchInput.Blur()
                }
                return m, nil
            }

        case "down":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTableWithSearch {
                m.searchCursor = (m.searchCursor + 1) % searchCursorCount
                if m.searchCursor == searchCursorInput {
                    return m, m.searchInput.Focus()
                } else {
                    m.searchInput.Blur()
                }
                return m, nil
            }

        case "left", "right":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTableWithSearch {
                m.searchCursor = searchCursorInput
                return m, m.searchInput.Focus()
            }
            return m, nil

        case " ", "space":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTableWithSearch && m.searchCursor > searchCursorInput {
                m.toggleCheckbox()
                return m, nil
            }
        }
    }

    if m.loadState == LoadStateLoaded {
        switch m.viewMode {
        case ViewModeTableWithSearch:
            // route to search input only if cursor is on input
            if m.searchCursor == searchCursorInput {
                m.searchInput, cmd = m.searchInput.Update(msg)
                cmds = append(cmds, cmd)
            }

        case ViewModeTable:
            m.table, cmd = m.table.Update(msg)
            cmds = append(cmds, cmd)

            if m.table.Cursor() != m.selectedIndex {
                m.selectedIndex = m.table.Cursor()
            }

        case ViewModeTableWithSplit:
            // only update the focused viewport
            if m.focusedViewport == ViewportFocusRequest {
                m.requestViewport, cmd = m.requestViewport.Update(msg)
                cmds = append(cmds, cmd)
            } else {
                m.responseViewport, cmd = m.responseViewport.Update(msg)
                cmds = append(cmds, cmd)
            }
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

    tableHeight := m.calculateTableHeight()

    m.table = table.New(
        table.WithColumns(m.columns),
        table.WithRows(m.rows),
        table.WithFocused(true),
        table.WithHeight(tableHeight),
        table.WithWidth(m.width),
    )

    m.table = ApplyTableStyles(m.table)
    m.adjustColumnWidths()
}

func (m *HARViewModel) updateTableDimensions() {
    tableHeight := m.calculateTableHeight()
    m.table.SetHeight(tableHeight)
    m.table.SetWidth(m.width)

    m.adjustColumnWidths()
}

func (m *HARViewModel) calculateTableHeight() int {
    tableHeight := m.height - tableVerticalPadding

    switch m.viewMode {
    case ViewModeTableWithSplit:
        tableHeight /= 2
    case ViewModeTableWithSearch:
        tableHeight = int(float64(m.height-tableVerticalPadding) * searchTableHeightRatio)
    }

    return tableHeight
}

func (m *HARViewModel) calculatePanelDimensions() (panelWidth, panelHeight int) {
    panelWidth = m.width / 2
    panelHeight = m.height/2 - ((tableVerticalPadding / 2) - 1)
    return panelWidth, panelHeight
}

func (m *HARViewModel) updateViewportDimensions() {
    panelWidth, panelHeight := m.calculatePanelDimensions()

    // subtract border width (2 chars per panel)
    viewportWidth := panelWidth - 2
    viewportHeight := panelHeight - 2

    if m.requestViewport.Width() == 0 {
        m.requestViewport = viewport.New(viewport.WithWidth(viewportWidth), viewport.WithHeight(viewportHeight))
        m.responseViewport = viewport.New(viewport.WithWidth(viewportWidth), viewport.WithHeight(viewportHeight))
    } else {
        m.requestViewport.SetWidth(viewportWidth)
        m.requestViewport.SetHeight(viewportHeight)
        m.responseViewport.SetWidth(viewportWidth)
        m.responseViewport.SetHeight(viewportHeight)
    }
}

func (m *HARViewModel) toggleSplitView() {
    if m.viewMode == ViewModeTable {
        m.viewMode = ViewModeTableWithSplit
        m.focusedViewport = ViewportFocusRequest // Reset focus to request when opening
        m.updateTableDimensions()
        m.updateViewportDimensions()
    } else {
        m.viewMode = ViewModeTable
        m.updateTableDimensions()
    }
}

func (m *HARViewModel) toggleViewportFocus() {
    if m.focusedViewport == ViewportFocusRequest {
        m.focusedViewport = ViewportFocusResponse
    } else {
        m.focusedViewport = ViewportFocusRequest
    }
}

// toggleSearchView switches between table-only mode and table-with-search mode.
// When entering search mode, the table height is adjusted to 70% of available space
// and the search input receives focus. The colorized table is cached to improve performance.
func (m *HARViewModel) toggleSearchView() tea.Cmd {
    if m.viewMode == ViewModeTable {
        m.viewMode = ViewModeTableWithSearch
        m.updateTableDimensions()
        // cache the colorized table to avoid re-rendering on every keystroke
        tableView := m.table.View()
        m.cachedColorizedTable = ColorizeHARTableOutput(tableView, m.table.Cursor(), m.rows)
        m.cachedTableCursor = m.table.Cursor()
        // reset search state and focus input
        m.searchCursor = searchCursorInput
        m.searchQuery = ""
        m.searchOptions = [3]bool{} // reset all checkboxes to unchecked
        m.searchInput.SetValue("")
        return m.searchInput.Focus()
    } else {
        m.viewMode = ViewModeTable
        m.updateTableDimensions()
        // invalidate cache when leaving search mode
        m.cachedColorizedTable = ""
        m.cachedTableCursor = -1
        // blur input
        m.searchInput.Blur()
        return nil
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

// Cleanup releases resources when the model is destroyed
func (m *HARViewModel) Cleanup() error {
    if m.streamer != nil {
        return m.streamer.Close()
    }
    return nil
}
