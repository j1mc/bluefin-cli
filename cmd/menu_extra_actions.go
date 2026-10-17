package cmd

import (
	tea "charm.land/bubbletea/v2"

	"github.com/tuna-os/bluefin-cli/internal/tui/app"
)

// Extra menu destinations register themselves with the shared palette registry.
// The build-specific menu files remain responsible for which destinations are
// available on the current build and platform.
func init() {
	for _, it := range extraMenuItems() {
		it := it
		app.Register(app.Action{
			ID: it.Value, Icon: it.Icon, Label: it.Label, Section: "Customize",
			Do: func() tea.Cmd { return extraMenuDo(it.Value) },
		})
	}
}
