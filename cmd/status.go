package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tuna-os/bluefin-cli/internal/status"
	"github.com/tuna-os/bluefin-cli/internal/tui/app"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show configuration status",
	Long:  `Display the current configuration status for shell experience, MOTD, and installed tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return status.Show(reportedVersion())
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	app.Register(app.Action{
		ID: "status", Icon: "📊", Label: "Show Status", Section: "Home",
		Do: statusScreenCmd,
	})
}

func statusScreenCmd() tea.Cmd {
	return app.Push(app.NewText("Status", func(width int) (string, error) {
		return status.Render(width, reportedVersion())
	}))
}
