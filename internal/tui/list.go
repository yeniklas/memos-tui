package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yeniklas/memos-tui/internal/model"
)

type listModel struct {
	memos          []model.Memo
	cursor         int
	offset         int
	tags           []string
	activeTag      string
	activeDate     string
	shortcuts      []model.Shortcut
	activeShortcut string // Name of active shortcut, "" = none
	showArchived   bool
	nextToken      string
	loading        bool
	width          int
	height         int
}

func newListModel() listModel {
	return listModel{}
}

func (m listModel) selected() *model.Memo {
	if len(m.memos) == 0 || m.cursor >= len(m.memos) {
		return nil
	}
	memo := m.memos[m.cursor]
	return &memo
}

func (m listModel) visibleHeight() int {
	h := m.height - 3 // border (2) + status (1)
	if h < 1 {
		h = 1
	}
	return h
}

func (m listModel) View(focused bool) string {
	inner := m.renderInner()
	style := stylePanel
	if focused {
		style = stylePanelFocused
	}
	return style.Width(m.width - 2).Height(m.height - 2).Render(inner)
}

func (m listModel) renderInner() string {
	var sb strings.Builder

	if m.loading && len(m.memos) == 0 {
		sb.WriteString(styleMuted.Render("Loading…"))
		return sb.String()
	}

	if len(m.memos) == 0 {
		sb.WriteString(styleMuted.Render("No memos."))
		return sb.String()
	}

	visible := m.visibleHeight()
	end := m.offset + visible
	if end > len(m.memos) {
		end = len(m.memos)
	}

	contentWidth := m.width - 4 // inside border + padding
	dateWidth := 10              // "2006-01-02"
	tagMaxWidth := 20
	// 3 chars reserved for pin indicator ("📌 " or "   ") before date
	snippetWidth := contentWidth - dateWidth - tagMaxWidth - 4 - 3

	// Count pinned memos (they are sorted first by sortPinnedFirst).
	var pinnedCount int
	for _, memo := range m.memos {
		if !memo.Pinned {
			break
		}
		pinnedCount++
	}

	for i := m.offset; i < end; i++ {
		memo := m.memos[i]
		isSelected := i == m.cursor

		prefix := "  "
		if isSelected {
			prefix = styleSelected.Render("> ")
		}

		date := memo.DisplayTime.Format("2006-01-02")
		snippet := truncate(memoTitle(memo.Content), snippetWidth)

		var row string
		switch {
		case memo.Pinned && !isSelected:
			tagsPlain := plainTags(memo.Tags, tagMaxWidth)
			content := fmt.Sprintf("📌 %s  %-*s  %-*s",
				date, snippetWidth, snippet, tagMaxWidth, tagsPlain)
			row = prefix + stylePinned.Render(content)
		case isSelected:
			pinPrefix := "   "
			if memo.Pinned {
				pinPrefix = "📌 "
			}
			tagsPlain := plainTags(memo.Tags, tagMaxWidth)
			content := fmt.Sprintf("%s%s  %-*s  %-*s",
				pinPrefix, date, snippetWidth, snippet, tagMaxWidth, tagsPlain)
			row = prefix + styleSelected.Render(content)
		default:
			dateStr := styleDate.Render(date)
			tagsStr := renderTagsInline(memo.Tags, tagMaxWidth)
			row = fmt.Sprintf("%s   %s  %-*s  %s",
				prefix, dateStr, snippetWidth, snippet, tagsStr)
		}

		sb.WriteString(row)
		sb.WriteString("\n")

		// Draw separator between last pinned memo and first unpinned memo.
		if i == pinnedCount-1 && pinnedCount < len(m.memos) {
			sb.WriteString(styleMuted.Render(strings.Repeat("─", max(m.width-4, 0))))
			sb.WriteString("\n")
		}
	}

	// scroll indicator
	if m.nextToken != "" || end < len(m.memos) {
		sb.WriteString(styleMuted.Render("  ↓ more"))
	}

	return sb.String()
}

// rootTagsFrom returns the unique first path segments of all tags, sorted.
func rootTagsFrom(tags []string) []string {
	seen := map[string]bool{}
	var roots []string
	for _, t := range tags {
		root := strings.SplitN(t, "/", 2)[0]
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	return roots
}

// hasSubtags reports whether any tag begins with root+"/".
func hasSubtags(tags []string, root string) bool {
	prefix := root + "/"
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// memoTitle returns the first line of content with any leading markdown header
// markers stripped. "# My note" becomes "My note". Lines starting with "#"
// but no following space (e.g. "#tag") are left unchanged.
func memoTitle(content string) string {
	line := firstLine(content)
	// Count leading '#' characters
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	// Not a markdown header: no '#', too many (>6), or '#' not followed by space/tab/EOL
	if i == 0 || i > 6 {
		return line
	}
	rest := line[i:]
	if rest == "" || rest[0] == ' ' || rest[0] == '\t' {
		return strings.TrimSpace(rest)
	}
	return line // e.g. "#tag" — not a header
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

func renderTagsInline(tags []string, maxWidth int) string {
	if len(tags) == 0 {
		return strings.Repeat(" ", maxWidth)
	}
	var parts []string
	for _, t := range tags {
		parts = append(parts, "#"+t)
	}
	joined := strings.Join(parts, " ")
	if utf8.RuneCountInString(joined) > maxWidth {
		joined = truncate(joined, maxWidth)
	}
	return styleTag.Render(fmt.Sprintf("%-*s", maxWidth, joined))
}

func plainTags(tags []string, maxWidth int) string {
	if len(tags) == 0 {
		return strings.Repeat(" ", maxWidth)
	}
	var parts []string
	for _, t := range tags {
		parts = append(parts, "#"+t)
	}
	joined := strings.Join(parts, " ")
	if utf8.RuneCountInString(joined) > maxWidth {
		joined = truncate(joined, maxWidth)
	}
	return fmt.Sprintf("%-*s", maxWidth, joined)
}

