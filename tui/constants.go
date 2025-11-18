package tui

const (
	tableVerticalPadding = 5 // title(2) + newlines(2) + footer(1)
	splitPanelPadding    = 2
	minURLColumnWidth    = 20
	maxURLColumnWidth    = 100
	borderPadding        = 6

	methodColumnWidth   = 8
	statusColumnWidth   = 10
	durationColumnWidth = 10

	// Search panel dimensions
	searchPanelHeightRatio = 0.3  // 30% of vertical space
	searchTableHeightRatio = 0.7  // 70% of vertical space
	minSearchPanelHeight   = 5    // minimum height in lines

	// Search cursor positions
	searchCursorInput = 0
	searchCursorOpt1  = 1
	searchCursorOpt2  = 2
	searchCursorOpt3  = 3
	searchCursorOpt4  = 4
	searchCursorCount = 5
)
