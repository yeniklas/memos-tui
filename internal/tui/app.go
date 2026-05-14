package tui

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yeniklas/memos-tui/internal/api"
	"github.com/yeniklas/memos-tui/internal/config"
	"github.com/yeniklas/memos-tui/internal/model"
)

type panel int

const (
	panelList panel = iota
	panelPreview
	panelSearch
	panelFilterPopup
	panelCalendar
)

type filterPopupModel struct {
	cursor int
	offset int // row scroll offset for long tag lists
}

type deleteState int

const (
	deleteNone deleteState = iota
	deleteConfirm
)

// App is the root bubbletea model.
type App struct {
	client      *api.Client
	focus       panel
	list        listModel
	preview     previewModel
	search      searchModel
	filterPopup filterPopupModel
	calendar    calendarModel
	width       int
	height      int
	err         error
	delState    deleteState
	spinner     spinner.Model
	searchMode  bool // true when displaying search results
	searchQuery string
	profile     string
	version     string
}

func NewApp(client *api.Client, markdownEnabled bool, theme config.Theme, profile, version string) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &App{
		client:  client,
		focus:   panelList,
		list:    newListModel(),
		preview: newPreviewModel(markdownEnabled, theme),
		search:  newSearchModel(),
		spinner: sp,
		profile: profile,
		version: version,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		loadMemosCmd(a.client, "", "", false),
		loadTagsCmd(a.client),
		a.spinner.Tick,
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.distributeSize()

	case tea.KeyMsg:
		return a.handleKey(msg)

	case memosLoadedMsg:
		a.list.loading = false
		a.list.memos = sortPinnedFirst(msg.memos)
		a.list.nextToken = msg.nextToken
		a.list.cursor = 0
		a.list.offset = 0
		a.updatePreview()
		// Load shortcuts once we know who the current user is (from memo creator field).
		if a.list.shortcuts == nil && len(msg.memos) > 0 {
			return a, loadShortcutsCmd(a.client, msg.memos[0].Creator)
		}

	case tagsLoadedMsg:
		a.list.tags = msg.tags

	case moreMemosLoadedMsg:
		a.list.loading = false
		a.list.memos = sortPinnedFirst(dedupMemos(append(a.list.memos, msg.memos...)))
		a.list.nextToken = msg.nextToken

	case memoSavedMsg:
		return a, a.reloadMemos()

	case memoDeletedMsg:
		a.delState = deleteNone
		// remove from local list immediately
		for i, m := range a.list.memos {
			if m.Name == msg.name {
				a.list.memos = append(a.list.memos[:i], a.list.memos[i+1:]...)
				break
			}
		}
		if a.list.cursor >= len(a.list.memos) && a.list.cursor > 0 {
			a.list.cursor--
		}
		a.updatePreview()

	case memoArchivedMsg:
		for i, m := range a.list.memos {
			if m.Name == msg.name {
				a.list.memos = append(a.list.memos[:i], a.list.memos[i+1:]...)
				break
			}
		}
		if a.list.cursor >= len(a.list.memos) && a.list.cursor > 0 {
			a.list.cursor--
		}
		a.updatePreview()

	case memoUpdatedMsg:
		selectedName := ""
		if sel := a.list.selected(); sel != nil {
			selectedName = sel.Name
		}
		for i, m := range a.list.memos {
			if m.Name == msg.memo.Name {
				a.list.memos[i] = msg.memo
				break
			}
		}
		a.list.memos = sortPinnedFirst(a.list.memos)
		if selectedName != "" {
			for i, m := range a.list.memos {
				if m.Name == selectedName {
					a.list.cursor = i
					a.adjustOffset()
					break
				}
			}
		}
		a.updatePreview()

	case shortcutsLoadedMsg:
		a.list.shortcuts = msg.shortcuts

	case calendarDaysMsg:
		if msg.year == a.calendar.year && msg.month == a.calendar.month {
			a.calendar.daysWithMemos = msg.days
			a.calendar.loading = false
		}

	case searchResultsMsg:
		a.search.loading = false
		a.list.loading = false
		a.list.memos = msg.memos
		a.list.nextToken = ""
		a.list.cursor = 0
		a.list.offset = 0
		a.updatePreview()

	case errMsg:
		a.list.loading = false
		a.search.loading = false
		a.err = msg.err

	case spinner.TickMsg:
		var cmd, searchCmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		a.search.spinner, searchCmd = a.search.spinner.Update(msg)
		return a, tea.Batch(cmd, searchCmd)
	}

	// forward viewport scrolling to preview when focused
	if a.focus == panelPreview {
		vp, cmd := a.preview.vp.Update(msg)
		a.preview.vp = vp
		return a, cmd
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// global quit
	if key := msg.String(); key == "ctrl+c" {
		return a, tea.Quit
	}
	// clear error on any keypress
	a.err = nil

	// filter popup
	if a.focus == panelFilterPopup {
		return a.handleFilterPopupKey(msg)
	}

	// calendar overlay
	if a.focus == panelCalendar {
		return a.handleCalendarKey(msg)
	}

	// search overlay active
	if a.focus == panelSearch {
		return a.handleSearchKey(msg)
	}

	switch {
	case keyMatches(msg, keys.Quit):
		return a, tea.Quit

	case keyMatches(msg, keys.Preview):
		if a.focus == panelList {
			a.focus = panelPreview
		} else {
			a.focus = panelList
		}

	case keyMatches(msg, keys.Search):
		a.focus = panelSearch
		a.search.input.SetValue("")
		a.search.input.Focus()

	case keyMatches(msg, keys.Filter):
		if a.focus == panelList {
			a.openFilterPopup()
		}

	case keyMatches(msg, keys.Calendar):
		if a.focus == panelList {
			a.openCalendar()
			return a, loadCalendarDaysCmd(a.client, a.currentFilter(), a.calendar.year, a.calendar.month)
		}

	case keyMatches(msg, keys.ClearFilters):
		if a.focus == panelList {
			a.list.activeTag = ""
			a.list.activeShortcut = ""
			a.list.activeDate = ""
			a.list.showArchived = false
			a.searchMode = false
			a.searchQuery = ""
			return a, a.reloadMemos()
		}

	case keyMatches(msg, keys.Up):
		a.moveCursor(-1)

	case keyMatches(msg, keys.Down):
		return a, a.moveCursorDown()

	case keyMatches(msg, keys.PageUp):
		a.moveCursor(-a.list.visibleHeight())
		a.updatePreview()

	case keyMatches(msg, keys.PageDown):
		a.moveCursor(a.list.visibleHeight())
		a.updatePreview()
		if a.list.cursor == len(a.list.memos)-1 && a.list.nextToken != "" && !a.list.loading {
			a.list.loading = true
			return a, loadMoreCmd(a.client, a.currentFilter(), a.list.nextToken, a.list.showArchived)
		}

	case keyMatches(msg, keys.Top):
		a.list.cursor = 0
		a.list.offset = 0
		a.updatePreview()

	case keyMatches(msg, keys.Bottom):
		a.list.cursor = len(a.list.memos) - 1
		a.adjustOffset()
		a.updatePreview()

	case keyMatches(msg, keys.New):
		return a, a.openEditor(nil)

	case keyMatches(msg, keys.Edit):
		if sel := a.list.selected(); sel != nil {
			return a, a.openEditor(sel)
		}

	case keyMatches(msg, keys.Delete):
		if sel := a.list.selected(); sel != nil {
			if a.delState == deleteConfirm {
				a.delState = deleteNone
				return a, deleteCmd(a.client, sel.Name)
			}
			a.delState = deleteConfirm
		}

	case keyMatches(msg, keys.Pin):
		if sel := a.list.selected(); sel != nil {
			return a, togglePinCmd(a.client, sel.Name, !sel.Pinned)
		}

	case keyMatches(msg, keys.Vis):
		if sel := a.list.selected(); sel != nil {
			next := sel.Visibility.Next()
			return a, updateVisCmd(a.client, sel.Name, next)
		}

	case keyMatches(msg, keys.Archive):
		if sel := a.list.selected(); sel != nil {
			newState := model.MemoStateArchived
			if a.list.showArchived {
				newState = model.MemoStateNormal
			}
			return a, archiveCmd(a.client, sel.Name, newState)
		}

	case keyMatches(msg, keys.Refresh):
		a.searchMode = false
		a.searchQuery = ""
		return a, a.reloadMemos()

	default:
		// any other key clears delete confirmation
		if a.delState == deleteConfirm {
			a.delState = deleteNone
		}
	}

	return a, nil
}

// filterTagList returns the ordered tag display list, inserting synthetic parent
// entries before their children so that each hierarchy root is selectable.
func filterTagList(tags []string) []string {
	roots := rootTagsFrom(tags)
	added := map[string]bool{}
	var result []string
	for _, root := range roots {
		if hasSubtags(tags, root) && !added[root] {
			result = append(result, root)
			added[root] = true
		}
		for _, t := range tags {
			if strings.SplitN(t, "/", 2)[0] == root && !added[t] {
				result = append(result, t)
				added[t] = true
			}
		}
	}
	return result
}

func (a *App) openFilterPopup() {
	sc := a.list.shortcuts
	displayTags := filterTagList(a.list.tags)
	a.filterPopup.cursor = 0
	a.filterPopup.offset = 0
	if a.list.showArchived {
		// cursor stays at 0 (the "Archived" entry)
	} else {
		for i, s := range sc {
			if s.Name == a.list.activeShortcut {
				a.filterPopup.cursor = 1 + i
				break
			}
		}
		if a.list.activeTag != "" {
			for i, tag := range displayTags {
				if tag == a.list.activeTag {
					a.filterPopup.cursor = 1 + len(sc) + i
					break
				}
			}
		}
	}
	a.focus = panelFilterPopup
}

func (a *App) handleFilterPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	displayTags := filterTagList(a.list.tags)
	nsc := len(a.list.shortcuts)
	total := 1 + nsc + len(displayTags)
	switch {
	case keyMatches(msg, keys.Up):
		if a.filterPopup.cursor > 0 {
			a.filterPopup.cursor--
		}
	case keyMatches(msg, keys.Down):
		if a.filterPopup.cursor < total-1 {
			a.filterPopup.cursor++
		}
	case keyMatches(msg, keys.Select):
		cursor := a.filterPopup.cursor
		if cursor == 0 {
			// Toggle "Archived" filter
			a.list.showArchived = !a.list.showArchived
			if a.list.showArchived {
				a.list.activeTag = ""
				a.list.activeShortcut = ""
			}
		} else if cursor <= nsc {
			s := a.list.shortcuts[cursor-1]
			if a.list.activeShortcut == s.Name {
				a.list.activeShortcut = ""
			} else {
				a.list.activeShortcut = s.Name
				a.list.activeTag = ""
				a.list.showArchived = false
			}
		} else {
			tag := displayTags[cursor-1-nsc]
			if a.list.activeTag == tag {
				a.list.activeTag = ""
			} else {
				a.list.activeTag = tag
				a.list.activeShortcut = ""
				a.list.showArchived = false
			}
		}
		a.focus = panelList
		return a, a.reloadMemos()
	case msg.String() == "esc":
		a.focus = panelList
	}
	return a, nil
}

func (a *App) openCalendar() {
	if a.calendar.year == 0 {
		a.calendar = newCalendarModel()
	}
	a.calendar.loading = true
	a.focus = panelCalendar
}

func (a *App) handleCalendarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m := &a.calendar
	dim := daysInMonth(m.year, m.month)

	changeMonth := func(delta int) tea.Cmd {
		m.month += time.Month(delta)
		if m.month < time.January {
			m.month = time.December
			m.year--
		} else if m.month > time.December {
			m.month = time.January
			m.year++
		}
		newDim := daysInMonth(m.year, m.month)
		if m.cursor > newDim {
			m.cursor = newDim
		}
		m.loading = true
		return loadCalendarDaysCmd(a.client, a.currentFilter(), m.year, m.month)
	}

	switch msg.String() {
	case "esc", "c":
		a.focus = panelList
	case "h", "left":
		m.cursor--
		if m.cursor < 1 {
			cmd := changeMonth(-1)
			m.cursor = daysInMonth(m.year, m.month)
			return a, cmd
		}
	case "l", "right":
		m.cursor++
		if m.cursor > dim {
			cmd := changeMonth(1)
			m.cursor = 1
			return a, cmd
		}
	case "k", "up":
		m.cursor -= 7
		if m.cursor < 1 {
			m.cursor = 1
		}
	case "j", "down":
		m.cursor += 7
		if m.cursor > dim {
			m.cursor = dim
		}
	case "[", "H":
		return a, changeMonth(-1)
	case "]", "L":
		return a, changeMonth(1)
	case "enter":
		a.list.activeDate = fmt.Sprintf("%d-%02d-%02d", m.year, int(m.month), m.cursor)
		a.focus = panelList
		return a, a.reloadMemos()
	}
	return a, nil
}

func (a *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.focus = panelList
		if a.searchMode {
			a.searchMode = false
			a.search.loading = false
			return a, a.reloadMemos()
		}
		return a, nil
	case "enter":
		q := strings.TrimSpace(a.search.input.Value())
		if q == "" {
			a.focus = panelList
			return a, nil
		}
		a.searchQuery = q
		a.searchMode = true
		a.search.loading = true
		a.list.loading = true
		a.focus = panelList
		return a, tea.Batch(
			searchCmd(a.client, q),
			a.search.spinner.Tick,
		)
	default:
		var cmd tea.Cmd
		a.search, cmd = a.search.Update(msg)
		return a, cmd
	}
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading…"
	}

	if a.focus == panelFilterPopup {
		return a.renderFilterPopupView()
	}

	if a.focus == panelCalendar {
		return a.calendar.render(a.width, a.height)
	}

	listView := a.list.View(a.focus == panelList)
	previewView := a.preview.View(a.focus == panelPreview)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, listView, previewView)

	statusBar := a.renderStatus()
	helpBar := a.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left,
		panels,
		statusBar,
		helpBar,
	)
}

func (a *App) renderFilterPopupView() string {
	const popupWidth = 32
	sep := styleMuted.Render(strings.Repeat("─", popupWidth-2))
	cursor := a.filterPopup.cursor
	nsc := len(a.list.shortcuts)
	displayTags := filterTagList(a.list.tags)

	var rows []string
	cursorRow := map[int]int{} // cursor index -> row index

	// "Archived" entry at position 0
	cursorRow[0] = len(rows)
	{
		label := "Archived"
		isActive := a.list.showArchived
		isCursor := cursor == 0
		switch {
		case isCursor && isActive:
			label = styleSelected.Render("> " + styleTagChipActive.Render(label))
		case isCursor:
			label = styleSelected.Render("> " + label)
		case isActive:
			label = "  " + styleTagChipActive.Render(label)
		default:
			label = "  " + label
		}
		rows = append(rows, label)
	}

	if nsc > 0 {
		rows = append(rows, "")
		rows = append(rows, styleSelected.Bold(true).Render(" Shortcuts "))
		rows = append(rows, sep)
		for i, s := range a.list.shortcuts {
			cursorRow[1+i] = len(rows)
			label := s.Title
			isActive := a.list.activeShortcut == s.Name
			isCursor := cursor == 1+i
			switch {
			case isCursor && isActive:
				label = styleSelected.Render("> " + styleTagChipActive.Render(label))
			case isCursor:
				label = styleSelected.Render("> " + label)
			case isActive:
				label = "  " + styleTagChipActive.Render(label)
			default:
				label = "  " + label
			}
			rows = append(rows, label)
		}
	}

	if len(displayTags) > 0 {
		rows = append(rows, "")
		rows = append(rows, styleSelected.Bold(true).Render(" Tags "))
		rows = append(rows, sep)
		for i, tag := range displayTags {
			cursorRow[1+nsc+i] = len(rows)
			isParent := hasSubtags(a.list.tags, tag)
			indent := "  "
			if isParent {
				indent = ""
			}
			label := indent + "#" + tag
			isActive := a.list.activeTag == tag
			isCursor := cursor == 1+nsc+i
			switch {
			case isCursor && isActive:
				label = styleSelected.Render("> " + styleTagChipActive.Render(strings.TrimLeft(label, " ")))
			case isCursor:
				label = styleSelected.Render("> " + strings.TrimLeft(label, " "))
			case isActive:
				label = indent + styleTagChipActive.Render("#"+tag)
			}
			rows = append(rows, label)
		}
	}

	rows = append(rows, "")
	rows = append(rows, styleMuted.Render("enter:select  esc:close"))

	// Cap visible rows to terminal height and scroll to keep cursor in view.
	// The box border adds 2 rows; leave a small margin.
	maxRows := a.height - 4
	if maxRows < 4 {
		maxRows = 4
	}
	offset := a.filterPopup.offset
	if row, ok := cursorRow[cursor]; ok {
		if row < offset {
			offset = row
		}
		if row >= offset+maxRows {
			offset = row - maxRows + 1
		}
	}
	a.filterPopup.offset = offset

	visible := rows
	if len(rows) > maxRows {
		end := offset + maxRows
		if end > len(rows) {
			end = len(rows)
		}
		visible = rows[offset:end]
	}

	content := strings.Join(visible, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Width(popupWidth).
		Render(content)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

func (a *App) renderStatus() string {
	if a.focus == panelSearch {
		return a.search.View()
	}

	var parts []string

	if a.list.loading {
		parts = append(parts, a.spinner.View()+" loading…")
	}
	if a.searchMode {
		parts = append(parts, styleMuted.Render(fmt.Sprintf("search: %q", a.searchQuery)))
	}
	if a.delState == deleteConfirm {
		parts = append(parts, styleError.Render("Press d again to confirm delete, any other key to cancel"))
	}
	if a.err != nil {
		parts = append(parts, styleError.Render("Error: "+a.err.Error()))
	}

	base := styleMuted.Render(fmt.Sprintf("%d memos", len(a.list.memos)))
	if indicator := a.renderFilterIndicator(); indicator != "" {
		base += "  " + indicator
	}
	parts = append(parts, base)

	left := strings.Join(parts, "  ")
	right := styleMuted.Render("profile: " + a.profile + "  " + a.version)
	pad := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func (a *App) renderFilterIndicator() string {
	if a.list.showArchived {
		return styleTagChipActive.Render("[Archived]")
	}
	if a.list.activeShortcut != "" {
		for _, s := range a.list.shortcuts {
			if s.Name == a.list.activeShortcut {
				return styleTagChipActive.Render("[" + s.Title + "]")
			}
		}
	}
	if a.list.activeTag != "" {
		return styleTag.Render("#" + a.list.activeTag)
	}
	if a.list.activeDate != "" {
		return styleMuted.Render(a.list.activeDate)
	}
	return ""
}

func (a *App) renderHelp() string {
	entries := []string{
		"j/k:move", "f:filter", "c:calendar", "C:clear", "/:search", "n:new", "e:edit", "d:delete",
		"p:pin", "v:vis", "a:archive", "tab:preview", "r:refresh", "q:quit",
	}
	return styleHelp.Render(strings.Join(entries, "  "))
}

func (a *App) distributeSize() {
	listW := a.width * 35 / 100
	previewW := a.width - listW

	panelH := a.height - 2 // status + help bars

	a.list.width = listW
	a.list.height = panelH

	a.preview.setSize(previewW, panelH)

	a.search.width = a.width
}

func (a *App) moveCursor(delta int) {
	a.list.cursor += delta
	if a.list.cursor < 0 {
		a.list.cursor = 0
	}
	if a.list.cursor >= len(a.list.memos) {
		a.list.cursor = len(a.list.memos) - 1
	}
	a.adjustOffset()
	a.updatePreview()
	a.delState = deleteNone
}

func (a *App) moveCursorDown() tea.Cmd {
	a.list.cursor++
	if a.list.cursor >= len(a.list.memos) {
		a.list.cursor = len(a.list.memos) - 1
		// load more if available
		if a.list.nextToken != "" && !a.list.loading {
			a.list.loading = true
			return loadMoreCmd(a.client, a.currentFilter(), a.list.nextToken, a.list.showArchived)
		}
	}
	a.adjustOffset()
	a.updatePreview()
	a.delState = deleteNone
	return nil
}

func (a *App) adjustOffset() {
	visible := a.list.visibleHeight()
	if a.list.cursor < a.list.offset {
		a.list.offset = a.list.cursor
	}
	if a.list.cursor >= a.list.offset+visible {
		a.list.offset = a.list.cursor - visible + 1
	}
}

func (a *App) updatePreview() {
	a.preview.setMemo(a.list.selected())
}

func (a *App) openEditor(memo *model.Memo) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	initial := ""
	if memo != nil {
		initial = memo.Content
	}

	tmp, err := os.CreateTemp("", "memos-tui-*.md")
	if err != nil {
		return func() tea.Msg { return errMsg{err} }
	}
	if initial != "" {
		if _, err := tmp.WriteString(initial); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return func() tea.Msg { return errMsg{err} }
		}
	}
	tmp.Close()

	tmpPath := tmp.Name()
	initialHash := hashFile(tmpPath)

	cmd := exec.Command(editor, tmpPath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			os.Remove(tmpPath)
			return errMsg{err}
		}
		content, readErr := os.ReadFile(tmpPath)
		os.Remove(tmpPath)
		if readErr != nil {
			return errMsg{readErr}
		}
		newHash := sha256.Sum256(content)
		if newHash == initialHash || strings.TrimSpace(string(content)) == "" {
			return nil // no change or empty
		}

		if memo == nil {
			m, apiErr := a.client.CreateMemo(string(content), model.VisibilityPrivate)
			if apiErr != nil {
				return errMsg{apiErr}
			}
			return memoSavedMsg{memo: m}
		}
		m, apiErr := a.client.UpdateMemo(memo.Name, string(content))
		if apiErr != nil {
			return errMsg{apiErr}
		}
		return memoSavedMsg{memo: m}
	})
}

func hashFile(path string) [32]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}
	}
	return sha256.Sum256(data)
}

// Commands

func loadMemosCmd(client *api.Client, filter, pageToken string, archived bool) tea.Cmd {
	return func() tea.Msg {
		memos, next, err := client.ListMemos(filter, pageToken, archived)
		if err != nil {
			return errMsg{err}
		}
		// On the first page of a non-archived load, prefetch all pinned memos so
		// they appear at the top regardless of the API's time-based sort order.
		if pageToken == "" && !archived {
			pinnedFilter := combineFilters(filter, "pinned == true")
			if pinned, _, err2 := client.ListMemos(pinnedFilter, "", false); err2 == nil && len(pinned) > 0 {
				seen := make(map[string]bool, len(pinned))
				for _, m := range pinned {
					seen[m.Name] = true
				}
				var rest []model.Memo
				for _, m := range memos {
					if !seen[m.Name] {
						rest = append(rest, m)
					}
				}
				memos = append(pinned, rest...)
			}
		}
		return memosLoadedMsg{memos: memos, nextToken: next}
	}
}

func loadMoreCmd(client *api.Client, filter, pageToken string, archived bool) tea.Cmd {
	return func() tea.Msg {
		memos, next, err := client.ListMemos(filter, pageToken, archived)
		if err != nil {
			return errMsg{err}
		}
		return moreMemosLoadedMsg{memos: memos, nextToken: next}
	}
}

func loadTagsCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		tags, err := client.ListTags()
		if err != nil {
			return errMsg{err}
		}
		sort.Strings(tags)
		return tagsLoadedMsg{tags: tags}
	}
}

func deleteCmd(client *api.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := client.DeleteMemo(name); err != nil {
			return errMsg{err}
		}
		return memoDeletedMsg{name: name}
	}
}

func archiveCmd(client *api.Client, name string, state model.MemoState) tea.Cmd {
	return func() tea.Msg {
		if _, err := client.UpdateState(name, state); err != nil {
			return errMsg{err}
		}
		return memoArchivedMsg{name: name}
	}
}

func togglePinCmd(client *api.Client, name string, pinned bool) tea.Cmd {
	return func() tea.Msg {
		m, err := client.UpdatePinned(name, pinned)
		if err != nil {
			return errMsg{err}
		}
		return memoUpdatedMsg{memo: m}
	}
}

func updateVisCmd(client *api.Client, name string, vis model.Visibility) tea.Cmd {
	return func() tea.Msg {
		m, err := client.UpdateVisibility(name, vis)
		if err != nil {
			return errMsg{err}
		}
		return memoUpdatedMsg{memo: m}
	}
}

// currentFilter returns the combined active shortcut + tag + state server-side filter.
// Date filtering is handled client-side (see loadMemosByDateCmd).
func (a *App) currentFilter() string {
	shortcutFilter := ""
	for _, s := range a.list.shortcuts {
		if s.Name == a.list.activeShortcut {
			shortcutFilter = s.Filter
			break
		}
	}
	return combineFilters(shortcutFilter, tagFilter(a.list.activeTag))
}

// reloadMemos returns a command that reloads the memo list, applying the
// active date filter client-side when one is set.
func (a *App) reloadMemos() tea.Cmd {
	a.list.loading = true
	if a.list.activeDate != "" {
		return loadMemosByDateCmd(a.client, a.currentFilter(), a.list.activeDate, a.list.showArchived)
	}
	return loadMemosCmd(a.client, a.currentFilter(), "", a.list.showArchived)
}

// loadMemosByDateCmd fetches all pages matching serverFilter and returns only
// memos whose displayTime (UTC) falls on date ("YYYY-MM-DD").
func loadMemosByDateCmd(client *api.Client, serverFilter, date string, archived bool) tea.Cmd {
	return func() tea.Msg {
		var all []model.Memo
		pageToken := ""
		for {
			memos, next, err := client.ListMemos(serverFilter, pageToken, archived)
			if err != nil {
				return errMsg{err}
			}
			all = append(all, memos...)
			if next == "" {
				break
			}
			pageToken = next
		}
		return memosLoadedMsg{memos: filterMemosByDate(all, date)}
	}
}

func filterMemosByDate(memos []model.Memo, date string) []model.Memo {
	var result []model.Memo
	for _, m := range memos {
		if m.DisplayTime.UTC().Format("2006-01-02") == date {
			result = append(result, m)
		}
	}
	return result
}

// combineFilters joins non-empty filters with " && ".
func combineFilters(filters ...string) string {
	var parts []string
	for _, f := range filters {
		if f != "" {
			parts = append(parts, f)
		}
	}
	return strings.Join(parts, " && ")
}

func tagFilter(tag string) string {
	if tag == "" {
		return ""
	}
	return fmt.Sprintf(`content.contains("#%s")`, tag)
}

func keyMatches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

func loadShortcutsCmd(client *api.Client, userName string) tea.Cmd {
	return func() tea.Msg {
		shortcuts, err := client.ListShortcuts(userName)
		if err != nil {
			return errMsg{err}
		}
		return shortcutsLoadedMsg{shortcuts: shortcuts}
	}
}

func dedupMemos(memos []model.Memo) []model.Memo {
	seen := make(map[string]bool, len(memos))
	result := make([]model.Memo, 0, len(memos))
	for _, m := range memos {
		if !seen[m.Name] {
			seen[m.Name] = true
			result = append(result, m)
		}
	}
	return result
}

func sortPinnedFirst(memos []model.Memo) []model.Memo {
	result := make([]model.Memo, 0, len(memos))
	var unpinned []model.Memo
	for _, m := range memos {
		if m.Pinned {
			result = append(result, m)
		} else {
			unpinned = append(unpinned, m)
		}
	}
	return append(result, unpinned...)
}
