package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"github.com/pb33f/testify/require"
)

func TestViewUsesAlternateScreen(t *testing.T) {
	m := &HARViewModel{loadState: LoadStateLoaded}
	view := m.View()
	require.True(t, view.AltScreen)
	require.Contains(t, view.Content, "Initializing...")
	m.quitting = true
	require.Empty(t, m.View().Content)
}

func TestDetailModalScrolling(t *testing.T) {
	m := &HARViewModel{activeModal: ModalRequestFull, detailSearchState: &ViewportSearchState{},
		detailViewport: viewport.New(viewport.WithWidth(40), viewport.WithHeight(5))}
	m.detailViewport.SetContent(strings.Repeat("line\n", 30))
	for _, step := range []struct {
		key    string
		offset int
	}{{"down", 1}, {"pgdown", 6}, {"up", 5}, {"pgup", 0}} {
		handled, _ := m.handleDetailModalKeys(step.key)
		require.True(t, handled)
		require.Equal(t, step.offset, m.detailViewport.YOffset(), step.key)
	}
}
