package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yeniklas/memos-tui/internal/api"
)

type searchModel struct {
	input   textinput.Model
	spinner spinner.Model
	loading bool
	width   int
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "search memos…"
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return searchModel{input: ti, spinner: sp}
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m searchModel) View() string {
	prefix := "/ "
	if m.loading {
		prefix = m.spinner.View() + " "
	}
	bar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Width(m.width - 4).
		Padding(0, 1).
		Render(prefix + m.input.View())
	return bar
}

func searchCmd(client *api.Client, query string) tea.Cmd {
	return func() tea.Msg {
		filter := `content.contains("` + query + `")`
		memos, _, err := client.ListMemos(filter, "", false)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{memos: memos}
	}
}
