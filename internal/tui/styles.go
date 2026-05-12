package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/yeniklas/memos-tui/internal/config"
)

var (
	colorPrimary  = lipgloss.Color("63")
	colorMuted    = lipgloss.Color("241")
	colorSelected = lipgloss.Color("212")
	colorDate     = lipgloss.Color("246")
	colorTag      = lipgloss.Color("79")
	colorError    = lipgloss.Color("160")
	colorPinned   = lipgloss.Color("220")

	stylePanel         lipgloss.Style
	stylePanelFocused  lipgloss.Style
	styleSelected      lipgloss.Style
	styleDate          lipgloss.Style
	styleTag           lipgloss.Style
	styleTagChipActive lipgloss.Style
	styleMuted         lipgloss.Style
	styleError         lipgloss.Style
	stylePinned        lipgloss.Style
	styleMetaBar       lipgloss.Style
	styleHelp          lipgloss.Style
)

func init() {
	ApplyTheme(config.DefaultTheme())
}

// ApplyTheme rebuilds all package-level styles from the given theme.
// Call this once before starting the TUI program.
func ApplyTheme(t config.Theme) {
	colorPrimary  = lipgloss.Color(t.Primary)
	colorMuted    = lipgloss.Color(t.Muted)
	colorSelected = lipgloss.Color(t.Selected)
	colorDate     = lipgloss.Color(t.Date)
	colorTag      = lipgloss.Color(t.Tag)
	colorError    = lipgloss.Color(t.Error)
	colorPinned   = lipgloss.Color(t.Pinned)

	stylePanel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted)

	stylePanelFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary)

	styleSelected = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)

	styleDate = lipgloss.NewStyle().
		Foreground(colorDate)

	styleTag = lipgloss.NewStyle().
		Foreground(colorTag)

	styleTagChipActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(colorTag).
		Padding(0, 1)

	styleMuted = lipgloss.NewStyle().
		Foreground(colorMuted)

	styleError = lipgloss.NewStyle().
		Foreground(colorError)

	stylePinned = lipgloss.NewStyle().
		Foreground(colorPinned)

	styleMetaBar = lipgloss.NewStyle().
		Foreground(colorMuted).
		Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
		Foreground(colorMuted)
}
