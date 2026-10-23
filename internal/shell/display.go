package shell

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

func printSuccess(format string, args ...any) {
	fmt.Println(successStyle.Render(fmt.Sprintf(format, args...)))
}

func printError(format string, args ...any) {
	fmt.Println(errorStyle.Render(fmt.Sprintf(format, args...)))
}

func printInfo(format string, args ...any) {
	fmt.Println(infoStyle.Render(fmt.Sprintf(format, args...)))
}
