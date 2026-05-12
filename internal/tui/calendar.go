package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yeniklas/memos-tui/internal/api"
)

type calendarModel struct {
	year          int
	month         time.Month
	cursor        int // selected day 1-31
	daysWithMemos map[int]bool
	loading       bool
}

func newCalendarModel() calendarModel {
	now := time.Now()
	return calendarModel{
		year:   now.Year(),
		month:  now.Month(),
		cursor: now.Day(),
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// mondayOffset returns the 0-indexed weekday of the 1st (Monday=0 … Sunday=6).
func mondayOffset(year int, month time.Month) int {
	wd := time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Weekday()
	return (int(wd) + 6) % 7
}

// calendarWidth is the lipgloss Width() value for the popup box.
// Content area = calendarWidth - 2 (padding 0,1 on each side) = 25 chars.
const calendarWidth = 27

func (m calendarModel) render(appWidth, appHeight int) string {
	contentW := calendarWidth - 2 // 25 chars

	// Month navigation header — centred
	monthStr := fmt.Sprintf("◀  %s %d  ▶", m.month.String(), m.year)
	header := lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center).Render(monthStr)

	// Weekday header
	weekHeader := " Mo Tu We Th Fr Sa Su"

	// Build day rows
	offset := mondayOffset(m.year, m.month)
	total := daysInMonth(m.year, m.month)
	now := time.Now()
	isCurrentMonth := now.Year() == m.year && now.Month() == m.month
	today := now.Day()

	var dayRows []string
	var row strings.Builder
	col := offset

	// Leading blank cells for the first row
	for i := 0; i < offset; i++ {
		row.WriteString("   ")
	}

	for day := 1; day <= total; day++ {
		isCursor := day == m.cursor
		isToday := isCurrentMonth && day == today
		hasEntry := m.daysWithMemos[day]

		cell := fmt.Sprintf("%2d", day)
		switch {
		case isCursor:
			cell = styleSelected.Render(cell)
		case hasEntry && isToday:
			cell = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(cell)
		case hasEntry:
			cell = lipgloss.NewStyle().Foreground(colorPrimary).Render(cell)
		case isToday:
			cell = lipgloss.NewStyle().Bold(true).Render(cell)
		}

		row.WriteString(" ")
		row.WriteString(cell)
		col++

		if col == 7 {
			dayRows = append(dayRows, row.String())
			row.Reset()
			col = 0
		}
	}
	if col > 0 {
		dayRows = append(dayRows, row.String())
	}

	// Loading or empty spacer
	var statusLine string
	if m.loading {
		statusLine = styleMuted.Render("  loading…")
	} else {
		statusLine = ""
	}

	helpLine := styleMuted.Render("h/l:month  j/k:week  enter:select")

	var lines []string
	lines = append(lines, header)
	lines = append(lines, weekHeader)
	lines = append(lines, dayRows...)
	lines = append(lines, statusLine)
	lines = append(lines, helpLine)

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Width(calendarWidth).
		Render(content)

	return lipgloss.Place(appWidth, appHeight, lipgloss.Center, lipgloss.Center, box)
}

// loadCalendarDaysCmd fetches all memos matching serverFilter and returns which
// days in year/month have at least one memo. Date comparison is client-side
// because the Memos API CEL environment does not expose displayTime.
func loadCalendarDaysCmd(client *api.Client, serverFilter string, year int, month time.Month) tea.Cmd {
	return func() tea.Msg {
		days := map[int]bool{}
		pageToken := ""
		for {
			memos, next, err := client.ListMemos(serverFilter, pageToken, false)
			if err != nil {
				break
			}
			for _, memo := range memos {
				d := memo.DisplayTime.UTC()
				if d.Year() == year && d.Month() == month {
					days[d.Day()] = true
				}
			}
			if next == "" {
				break
			}
			pageToken = next
		}
		return calendarDaysMsg{year: year, month: month, days: days}
	}
}
