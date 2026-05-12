package tui

import (
	"time"

	"github.com/yeniklas/memos-tui/internal/model"
)

type memosLoadedMsg struct {
	memos     []model.Memo
	nextToken string
}

type moreMemosLoadedMsg struct {
	memos     []model.Memo
	nextToken string
}

type memoSavedMsg struct {
	memo model.Memo
}

type memoDeletedMsg struct {
	name string
}

type memoArchivedMsg struct {
	name string
}

type memoUpdatedMsg struct {
	memo model.Memo
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type searchResultsMsg struct {
	memos []model.Memo
}

type shortcutsLoadedMsg struct {
	shortcuts []model.Shortcut
}

type tagsLoadedMsg struct {
	tags []string
}

type calendarDaysMsg struct {
	year  int
	month time.Month
	days  map[int]bool
}
