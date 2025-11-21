package tui

import (
    "context"
    "time"

    "github.com/charmbracelet/bubbles/v2/progress"
    "github.com/charmbracelet/bubbles/v2/spinner"
    "github.com/charmbracelet/bubbles/v2/table"
    "github.com/charmbracelet/bubbles/v2/textinput"
    "github.com/charmbracelet/bubbles/v2/viewport"
    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/charmbracelet/lipgloss/v2"
    "github.com/pb33f/harific/motor"
    "github.com/pb33f/harhar"
)

// ViewMode represents the different view states
type ViewMode int

const (
    ViewModeTable ViewMode = iota
    ViewModeTableWithSplit
    ViewModeTableWithSearch
    ViewModeTableFiltered // viewing filtered results without search panel
)

// ViewportFocus represents which viewport has focus
type ViewportFocus int

const (
    ViewportFocusRequest ViewportFocus = iota
    ViewportFocusResponse
)

// ModalType represents which modal is currently open
type ModalType int

const (
    ModalNone ModalType = iota
    ModalFileTypeFilter
    ModalRequestFull
    ModalResponseFull
)

// Search messages for async search execution
type searchDebounceMsg struct {
    id int64
}

type detailSearchDebounceMsg struct {
    id int64
}

type searchStartMsg struct{}

type searchResultsMsg struct {
    matches []motor.SearchResult
}

type searchCompleteMsg struct{}

type searchErrorMsg struct {
    err error
}

type HARViewModel struct {
    table           table.Model
    allEntries      []*motor.EntryMetadata
    rows            []table.Row
    columns         []table.Column
    filteredIndices []int // maps filtered table row position to original entry index

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
    searchOptions [4]bool // checkbox states: 0=ResponseBody, 1=Regex, 2=AllMatches, 3=LiveSearch
    searchCursor  int     // focus position: 0=input, 1-4=checkboxes

    // search engine
    searcher      *motor.HARSearcher
    reader        motor.EntryReader
    searchFilter  *SearchFilter
    filterChain   *FilterChain
    isSearching   bool
    searchSpinner spinner.Model
    searchCtx     context.Context
    searchCancel  context.CancelFunc
    debounceID    int64 // increments on each keystroke to cancel stale debounces

    // file type filter modal
    activeModal      ModalType
    fileTypeFilter   *FileTypeFilter
    filterCheckboxes [6]bool // Graphics, JS, CSS, Fonts, Markup, AllFiles
    filterCursor     int     // which checkbox is focused in modal

    // detail viewport modal (full request/response view)
    detailViewport viewport.Model
    detailViewType string // "request" or "response"

    // cache for colorized table during search mode
    cachedColorizedTable string
    cachedTableCursor    int

    // detail modal search state
    detailSearchState    *ViewportSearchState
    detailDebounceID     int64 // increments on each keystroke to cancel stale debounces

    fileName string

    loadState       LoadState
    loadingSpinner  spinner.Model
    indexingMessage string
    indexingTime    time.Duration
    progressBar     progress.Model
    progressChan    chan motor.IndexProgress
    indexingPercent float64
    indexingEntries int

    err error
}

func NewHARViewModel(fileName string) (*HARViewModel, error) {
    columns := []table.Column{
        {Title: "Method", Width: methodColumnWidth},
        {Title: "URL", Width: 50}, // Initial width, will be adjusted dynamically
        {Title: "Status", Width: statusColumnWidth},
        {Title: "Size", Width: sizeColumnWidth},
        {Title: "Duration", Width: durationColumnWidth},
    }

    searchInput := textinput.New()
    searchInput.CharLimit = 200

    searchSpinner := spinner.New()
    searchSpinner.Spinner = spinner.Dot
    searchSpinner.Style = lipgloss.NewStyle().Foreground(RGBPink)

    progressBar := progress.New(
        progress.WithDefaultScaledGradient(),
        progress.WithWidth(50),
    )

    m := &HARViewModel{
        fileName:         fileName,
        columns:          columns,
        viewMode:         ViewModeTable,
        selectedIndex:    0,
        focusedViewport:  ViewportFocusRequest,
        loadState:        LoadStateLoading,
        loadingSpinner:   createLoadingSpinner(),
        indexingMessage:  "Building index...",
        searchInput:      searchInput,
        searchCursor:     searchCursorInput,
        searchFilter:     NewSearchFilter(),
        filterChain:      NewFilterChain(),
        searchSpinner:    searchSpinner,
        activeModal:         ModalNone,
        fileTypeFilter:      NewFileTypeFilter(),
        filterCheckboxes:    [6]bool{true, true, true, true, true, true}, // all enabled by default
        progressBar:         progressBar,
        progressChan:        make(chan motor.IndexProgress, 10),
        detailSearchState:   NewViewportSearchState(),
    }

    return m, nil
}

func (m *HARViewModel) toggleCheckbox() {
    if m.searchCursor > searchCursorInput && m.searchCursor < searchCursorCount {
        m.searchOptions[m.searchCursor-1] = !m.searchOptions[m.searchCursor-1]
    }
}

func (m *HARViewModel) executeSearch() tea.Cmd {
    // safety check
    if m.searcher == nil {
        return nil
    }

    query := m.searchInput.Value()

    // cancel previous search
    if m.searchCancel != nil {
        m.searchCancel()
    }

    // build search options from checkboxes
    opts := motor.DefaultSearchOptions
    opts.SearchResponseBody = m.searchOptions[0] // Response Bodies
    opts.FirstMatchOnly = !m.searchOptions[2]    // All Matches (inverted)

    if m.searchOptions[1] {
        opts.Mode = motor.Regex // Regex Mode
    } else {
        opts.Mode = motor.PlainText
    }

    // create new search context
    ctx, cancel := context.WithCancel(context.Background())
    m.searchCtx = ctx
    m.searchCancel = cancel

    // capture query for the Cmd closure
    searchQuery := query

    // start search in background
    return func() tea.Msg {
        resultsChan, err := m.searcher.Search(ctx, searchQuery, opts)
        if err != nil {
            return searchErrorMsg{err: err}
        }

        // collect all results
        var allMatches []motor.SearchResult
        for batch := range resultsChan {
            allMatches = append(allMatches, batch...)
        }

        // always return results message (even if empty)
        return searchResultsMsg{matches: allMatches}
    }
}

func (m *HARViewModel) startDebounceTimer() tea.Cmd {
    m.debounceID++
    currentID := m.debounceID

    return func() tea.Msg {
        // 300ms debounce for live search - gives users time to finish typing
        time.Sleep(300 * time.Millisecond)
        return searchDebounceMsg{id: currentID}
    }
}

func (m *HARViewModel) startDetailDebounceTimer() tea.Cmd {
    m.detailDebounceID++
    currentID := m.detailDebounceID

    return func() tea.Msg {
        time.Sleep(200 * time.Millisecond)
        return detailSearchDebounceMsg{id: currentID}
    }
}

func (m *HARViewModel) applyFilters() {
    m.filterChain.Clear()

    if m.searchFilter.IsActive() {
        m.filterChain.Add(m.searchFilter)
    }

    if m.fileTypeFilter.IsActive() {
        m.filterChain.Add(m.fileTypeFilter)
    }

    // future filters added here
    // if m.methodFilter.IsActive() { m.filterChain.Add(m.methodFilter) }

    filteredRows, indices := m.filterChain.BuildFilteredRows(m.allEntries, m.rows)
    m.table.SetRows(filteredRows)
    m.filteredIndices = indices

    // invalidate colorized table cache when filters change
    m.cachedColorizedTable = ""
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

        // initialize reader and searcher
        reader, err := motor.NewEntryReader(m.fileName, msg.index)
        if err != nil {
            m.err = err
            m.loadState = LoadStateError
            return m, nil
        }
        m.reader = reader
        m.searcher = motor.NewSearcher(msg.streamer, reader)

        if m.width > 0 && m.height > 0 {
            m.initializeTable()
            m.ready = true
        }
        return m, nil

    case indexErrorMsg:
        m.loadState = LoadStateError
        m.err = msg.err
        return m, nil

    case indexProgressMsg:
        if msg.totalBytes > 0 {
            m.indexingPercent = float64(msg.bytesRead) / float64(msg.totalBytes)
        }
        m.indexingEntries = msg.entriesSoFar
        // continue listening for more progress (recursive command pattern)
        return m, m.listenForProgress()

    case searchDebounceMsg:
        // only execute if this is the latest debounce (not stale)
        if msg.id == m.debounceID {
            return m, func() tea.Msg { return searchStartMsg{} }
        }
        return m, nil

    case detailSearchDebounceMsg:
        // only execute if this is the latest detail debounce (not stale)
        if msg.id == m.detailDebounceID {
            // Execute the search now
            if m.detailSearchState != nil && m.detailSearchState.active && !m.detailSearchState.locked {
                // Update the query and perform search
                m.detailSearchState.query = m.detailSearchState.searchInput.Value()
                if m.detailSearchState.renderer != nil {
                    m.detailSearchState.performSearch()
                }
                m.updateDetailContent()
            }
        }
        return m, nil

    case searchStartMsg:
        query := m.searchInput.Value()

        // Only clear filter if we're actively in search mode with empty query
        // Don't clear if we're in filtered view (user pressed Esc to view results)
        if query == "" {
            if m.viewMode == ViewModeTableWithSearch {
                // User cleared the search input while in search mode
                m.searchQuery = ""
                m.searchFilter.Clear()
                m.applyFilters()
            }
            // If not in search mode, don't do anything (preserves filtered results)
            return m, nil
        }

        m.searchQuery = query  // Store the active search query
        m.isSearching = true
        // DON'T clear the filter yet - wait for results
        // This prevents the filter from becoming inactive between searches
        return m, tea.Batch(m.executeSearch(), m.searchSpinner.Tick)

    case searchResultsMsg:
        // Clear previous results and add new ones
        // This ensures we always show results from the latest search
        m.searchFilter.ClearMatches()  // Just clear matches, keep filter active
        m.searchFilter.SetSearched(true)
        for _, result := range msg.matches {
            if result.Error == nil {
                m.searchFilter.AddMatch(result.Index)
            }
        }
        m.isSearching = false
        m.applyFilters()
        return m, nil

    case searchCompleteMsg:
        m.searchFilter.SetSearched(true)
        m.isSearching = false
        m.applyFilters()
        return m, nil

    case searchErrorMsg:
        m.isSearching = false
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
        key := msg.String()

        // modal keys have priority (check detail modal first, then filter modal, then viewport search)
        if handled, cmd := m.handleDetailModalKeys(key); handled {
            return m, cmd
        }
        if handled, cmd := m.handleFilterModalKeys(key); handled {
            return m, cmd
        }

        switch key {
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

        case "s", "ctrl+f", "/":
            if m.loadState == LoadStateLoaded {
                // Otherwise activate global search (not for "/" in split mode)
                if (m.viewMode == ViewModeTable || m.viewMode == ViewModeTableFiltered) && key != "/" {
                    cmd := m.toggleSearchView()
                    return m, cmd
                }
            }
            // don't return - let 's' fall through to input handling in search mode

        case "enter", "return":
            if m.loadState == LoadStateLoaded {
                if m.viewMode == ViewModeTable || m.viewMode == ViewModeTableFiltered {
                    // In table or filtered mode, Enter opens split view
                    m.toggleSplitView()
                    if m.viewMode == ViewModeTableWithSplit {
                        if err := m.loadSelectedEntry(); err != nil {
                            m.err = err
                        }
                    }
                } else if m.viewMode == ViewModeTableWithSearch {
                    if m.searchCursor == searchCursorInput {
                        // In search mode on input, Enter triggers search immediately
                        m.debounceID++ // invalidate any pending debounce timers
                        return m, func() tea.Msg { return searchStartMsg{} }
                    } else {
                        // On checkbox, Enter toggles like Space
                        m.toggleCheckbox()
                    }
                } else if m.viewMode == ViewModeTableWithSplit {
                    // In split view, Enter opens detail modal for focused panel
                    if m.focusedViewport == ViewportFocusRequest {
                        m.activeModal = ModalRequestFull
                        m.detailViewType = "request"
                        m.detailViewport.GotoTop() // reset scroll when opening
                        m.detailSearchState.Clear() // clear any previous search
                    } else {
                        m.activeModal = ModalResponseFull
                        m.detailViewType = "response"
                        m.detailViewport.GotoTop() // reset scroll when opening
                        m.detailSearchState.Clear() // clear any previous search
                    }
                }
            }
            return m, nil

        case "esc":
            if m.loadState == LoadStateLoaded {
                if m.viewMode == ViewModeTableWithSearch {
                    // first Esc: close search panel, keep filters (go to Filtered mode)
                    m.viewMode = ViewModeTableFiltered
                    m.updateTableDimensions()
                    m.debounceID++ // cancel pending timers
                    m.searchInput.Blur()
                    return m, nil
                } else if m.viewMode == ViewModeTableFiltered {
                    // second Esc: clear filters, return to full table
                    m.viewMode = ViewModeTable
                    if m.searchCancel != nil {
                        m.searchCancel()
                        m.searchCancel = nil
                    }
                    m.searchQuery = ""  // Clear the search query
                    m.searchFilter.Clear()
                    m.applyFilters()
                    m.updateTableDimensions()  // Update table height when returning to normal view
                    return m, nil
                } else if m.viewMode == ViewModeTableWithSplit {
                    // Esc in split view: return to filtered or table based on active filters
                    if m.searchFilter.IsActive() {
                        m.viewMode = ViewModeTableFiltered
                    } else {
                        m.viewMode = ViewModeTable
                    }
                    m.updateTableDimensions()
                    return m, nil
                }
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

        case "f":
            // lowercase f: blocked in search mode (would type 'f' in input)
            if m.loadState == LoadStateLoaded && m.viewMode != ViewModeTableWithSearch {
                m.activeModal = ModalFileTypeFilter
                m.filterCursor = 0
                return m, nil
            }
            // in search mode, let 'f' fall through to input

        case "F": // Shift+F
            // Shift+F opens filter modal from ANY mode (including search)
            if m.loadState == LoadStateLoaded {
                m.activeModal = ModalFileTypeFilter
                m.filterCursor = 0
                return m, nil
            }

        case " ", "space":
            if m.loadState == LoadStateLoaded && m.viewMode == ViewModeTableWithSearch && m.searchCursor > searchCursorInput {
                m.toggleCheckbox()
                return m, nil
            }
        }
    }

    // Handle detail modal search input
    if m.activeModal == ModalRequestFull || m.activeModal == ModalResponseFull {
        if m.detailSearchState.active && m.detailSearchState.cursor == 0 {
            switch msg.(type) {
            case tea.KeyPressMsg:
                oldValue := m.detailSearchState.searchInput.Value()
                m.detailSearchState.searchInput, cmd = m.detailSearchState.searchInput.Update(msg)
                cmds = append(cmds, cmd)

                // Start debounce timer if value changed
                if m.detailSearchState.searchInput.Value() != oldValue {
                    cmds = append(cmds, m.startDetailDebounceTimer())
                }
                return m, tea.Batch(cmds...)
            }
        }
    }

    if m.loadState == LoadStateLoaded {
        switch m.viewMode {
        case ViewModeTableWithSearch:
            // update search spinner if searching
            if m.isSearching {
                m.searchSpinner, cmd = m.searchSpinner.Update(msg)
                cmds = append(cmds, cmd)
            }

            // route to search input only if cursor is on input
            if m.searchCursor == searchCursorInput {
                oldValue := m.searchInput.Value()
                m.searchInput, cmd = m.searchInput.Update(msg)
                cmds = append(cmds, cmd)

                // check if live search is enabled (checkbox 4)
                if m.searchOptions[3] && m.searchInput.Value() != oldValue {
                    // start debounce timer
                    cmds = append(cmds, m.startDebounceTimer())
                }
            }

        case ViewModeTableFiltered:
            // in filtered mode, allow table navigation
            m.table, cmd = m.table.Update(msg)
            cmds = append(cmds, cmd)

            if m.table.Cursor() != m.selectedIndex {
                m.selectedIndex = m.table.Cursor()
            }

        case ViewModeTable:
            m.table, cmd = m.table.Update(msg)
            cmds = append(cmds, cmd)

            if m.table.Cursor() != m.selectedIndex {
                m.selectedIndex = m.table.Cursor()
            }

        case ViewModeTableWithSplit:
            // Handle messages based on type
            switch msg := msg.(type) {
            case tea.KeyPressMsg:
                key := msg.String()

                // Pass scrolling keys to viewport
                switch key {
                case "up", "down", "pgup", "pgdown", "home", "end":
                        // Pass scrolling keys to viewport
                        if m.focusedViewport == ViewportFocusRequest {
                            m.requestViewport, cmd = m.requestViewport.Update(msg)
                            cmds = append(cmds, cmd)
                        } else {
                            m.responseViewport, cmd = m.responseViewport.Update(msg)
                            cmds = append(cmds, cmd)
                        }
                }
                // Other keys are handled by our key handler above
            default:
                // Pass non-keyboard messages to viewport (mouse events, etc.)
                if m.focusedViewport == ViewportFocusRequest {
                    m.requestViewport, cmd = m.requestViewport.Update(msg)
                    cmds = append(cmds, cmd)
                } else {
                    m.responseViewport, cmd = m.responseViewport.Update(msg)
                    cmds = append(cmds, cmd)
                }
            }
        }
    }

    return m, tea.Batch(cmds...)
}

func (m *HARViewModel) View() string {
    if m.quitting {
        return ""
    }

    var baseView string

    switch m.loadState {
    case LoadStateLoading:
        baseView = m.renderLoadingView()
    case LoadStateError:
        baseView = m.renderErrorView()
    case LoadStateLoaded:
        if !m.ready {
            baseView = "Initializing..."
        } else {
            baseView = m.render()
        }
    default:
        baseView = "Unknown state"
    }

    // create layers for modal system
    layers := []*lipgloss.Layer{
        lipgloss.NewLayer(baseView),
    }

    // add modal zlayer if active
    if m.activeModal != ModalNone {
        modal := m.renderActiveModal()
        if modal != "" {
            var x, y int
            if m.activeModal == ModalRequestFull || m.activeModal == ModalResponseFull {
                x, y = m.calculateDetailModalPosition()
            } else {
                x, y = m.calculateModalPosition()
            }
            layers = append(layers, lipgloss.NewLayer(modal).X(x).Y(y).Z(1))
        }
    }

    canvas := lipgloss.NewCanvas(layers...)
    return canvas.Render()
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
    // When returning to normal table view, recreate the table to ensure proper height
    // The Bubbles table component doesn't always properly recalculate when using SetHeight()
    if m.viewMode == ViewModeTable || m.viewMode == ViewModeTableFiltered {
        // Save current cursor position
        currentCursor := m.table.Cursor()

        // Get the current rows from the table (which may be filtered)
        currentRows := m.table.Rows()

        // Recreate the table with the correct height
        tableHeight := m.calculateTableHeight()
        m.table = table.New(
            table.WithColumns(m.columns),
            table.WithRows(currentRows),  // Use current rows, not m.rows!
            table.WithFocused(true),
            table.WithHeight(tableHeight),
            table.WithWidth(m.width),
        )

        m.table = ApplyTableStyles(m.table)
        m.adjustColumnWidths()

        // Restore cursor position
        m.table.SetCursor(currentCursor)
    } else {
        // For other modes, just update dimensions normally
        tableHeight := m.calculateTableHeight()
        m.table.SetHeight(tableHeight)
        m.table.SetWidth(m.width)
        m.adjustColumnWidths()
        m.table.UpdateViewport()
    }
}

func (m *HARViewModel) calculateTableHeight() int {
    switch m.viewMode {
    case ViewModeTableWithSplit:
        // Split view layout:
        // - Title with border: 2 rows
        // - Table: (height - 5) / 2
        // - Split panel: (height - 5) / 2
        // - Status bar: 1 row
        // - Newlines between sections: 2
        // Total overhead: 5 rows
        return (m.height - 2) / 2
    case ViewModeTableWithSearch:
        // Search view layout:
        // - Title with border: 2 rows
        // - Table: 70% of remaining
        // - Search panel: 30% of remaining
        // - Status bar: 1 row
        // - Newlines: 2
        // Total overhead: 5 rows
        availableHeight := m.height - 2
        return int(float64(availableHeight) * searchTableHeightRatio)
    default:
        // Normal table view:
        // - Title with border: 2 rows
        // - Table: rest
        // - Status bar: 1 row
        // - Newlines: 2
        // Using tableVerticalPadding constant which is 5
        return m.height - tableVerticalPadding
    }
}

func (m *HARViewModel) calculatePanelDimensions() (panelWidth, panelHeight int) {
    panelWidth = m.width / 2
    // Panel height matches table height in split view
    panelHeight = (m.height - 5) / 2
    return panelWidth, panelHeight
}

func (m *HARViewModel) calculateSearchPanelHeight() int {
    // Search panel gets 30% of available content space
    availableHeight := m.height - 5
    return int(float64(availableHeight) * searchPanelHeightRatio)
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
    if m.viewMode == ViewModeTable || m.viewMode == ViewModeTableFiltered {
        m.viewMode = ViewModeTableWithSplit
        m.focusedViewport = ViewportFocusRequest // Reset focus to request when opening
        m.updateTableDimensions()
        m.updateViewportDimensions()
    } else {
        // return to filtered mode if filters are active, otherwise table
        if m.searchFilter.IsActive() {
            m.viewMode = ViewModeTableFiltered
        } else {
            m.viewMode = ViewModeTable
        }
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


func (m *HARViewModel) calculateModalPosition() (int, int) {
    // Fixed modal width to match renderFilterModal
    modalWidth := 30

    // position on right with padding (for filter modal)
    rightPadding := 2
    x := m.width - modalWidth - rightPadding

    // position near top with small padding
    topPadding := 2
    y := topPadding

    if x < 0 {
        x = 0
    }
    if y < 0 {
        y = 0
    }

    return x, y
}

func (m *HARViewModel) calculateDetailModalPosition() (int, int) {
    modalWidth := int(float64(m.width) * 0.9)
    modalHeight := int(float64(m.height) * 0.9)

    // center horizontally
    x := (m.width - modalWidth) / 2

    // center vertically
    y := (m.height - modalHeight) / 2

    if x < 0 {
        x = 0
    }
    if y < 0 {
        y = 0
    }

    return x, y
}

func (m *HARViewModel) renderActiveModal() string {
    switch m.activeModal {
    case ModalFileTypeFilter:
        return m.renderFilterModal()
    case ModalRequestFull, ModalResponseFull:
        return m.renderDetailModal()
    default:
        return ""
    }
}

// toggleSearchView opens search mode from table or filtered mode.
// When entering search mode, the table height is adjusted to 70% of available space
// and the search input receives focus. The colorized table is cached to improve performance.
func (m *HARViewModel) toggleSearchView() tea.Cmd {
    m.viewMode = ViewModeTableWithSearch
    m.updateTableDimensions()
    // cache the colorized table to avoid re-rendering on every keystroke
    tableView := m.table.View()
    m.cachedColorizedTable = ColorizeHARTableOutput(tableView, m.table.Cursor(), m.rows)
    m.cachedTableCursor = m.table.Cursor()
    // reset search state and focus input
    m.searchCursor = searchCursorInput
    m.searchQuery = ""
    m.searchOptions = [4]bool{false, false, false, true} // Live Search ON by default
    m.searchInput.SetValue("")
    return m.searchInput.Focus()
}

func (m *HARViewModel) loadSelectedEntry() error {
    // Get the actual entry index, accounting for filtering
    actualIndex := m.selectedIndex
    if len(m.filteredIndices) > 0 && m.selectedIndex < len(m.filteredIndices) {
        actualIndex = m.filteredIndices[m.selectedIndex]
    }

    if actualIndex >= len(m.allEntries) {
        return nil
    }

    ctx := context.Background()
    entry, err := m.streamer.GetEntry(ctx, actualIndex)
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
    urlWidth := m.width - methodColumnWidth - statusColumnWidth - sizeColumnWidth - durationColumnWidth - borderPadding
    if urlWidth < minURLColumnWidth {
        urlWidth = minURLColumnWidth
    }

    m.columns[0].Width = methodColumnWidth
    m.columns[1].Width = urlWidth
    m.columns[2].Width = statusColumnWidth
    m.columns[3].Width = sizeColumnWidth
    m.columns[4].Width = durationColumnWidth

    m.table.SetColumns(m.columns)
}

// Cleanup releases resources when the model is destroyed
func (m *HARViewModel) Cleanup() error {
    // cancel any active search
    m.debounceID++ // invalidate pending debounce timers
    if m.searchCancel != nil {
        m.searchCancel()
    }

    // close reader
    if m.reader != nil {
        if err := m.reader.Close(); err != nil {
            return err
        }
    }

    // close streamer
    if m.streamer != nil {
        return m.streamer.Close()
    }
    return nil
}
