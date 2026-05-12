package tui

import "github.com/charmbracelet/bubbles/key"

type listKeys struct {
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Top          key.Binding
	Bottom       key.Binding
	Filter       key.Binding
	ClearFilters key.Binding
	Calendar     key.Binding
	Select       key.Binding
	Preview      key.Binding
	Search       key.Binding
	New          key.Binding
	Edit         key.Binding
	Delete       key.Binding
	Pin          key.Binding
	Vis          key.Binding
	Archive      key.Binding
	Refresh      key.Binding
	Quit         key.Binding
}

var keys = listKeys{
	Up:           key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
	Down:         key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
	PageUp:       key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
	PageDown:     key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "page down")),
	Top:          key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:       key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
	Filter:       key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
	ClearFilters: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "clear")),
	Calendar:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "calendar")),
	Select:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Preview:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "preview")),
	Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	New:          key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Pin:          key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pin")),
	Vis:          key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "visibility")),
	Archive:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
	Refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
