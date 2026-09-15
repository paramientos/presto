package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

var renderer = lipgloss.NewRenderer(os.Stderr)

var (
	boldStyle   = renderer.NewStyle().Bold(true)
	dimStyle    = renderer.NewStyle().Faint(true)
	addStyle    = renderer.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	dropStyle   = renderer.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	warnStyle   = renderer.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	errorStyle  = renderer.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	accentStyle = renderer.NewStyle().Foreground(lipgloss.Color("6"))
	cursorStyle = renderer.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
)

func Bold(s string) string { return boldStyle.Render(s) }

func Dim(s string) string { return dimStyle.Render(s) }

func Accent(s string) string { return accentStyle.Render(s) }
