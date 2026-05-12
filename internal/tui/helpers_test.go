package tui

import (
	"testing"

	"github.com/yeniklas/memos-tui/internal/model"
)

// --- firstLine ---

func TestFirstLine_singleLine(t *testing.T) {
	got := firstLine("Hello world")
	if got != "Hello world" {
		t.Errorf("got %q", got)
	}
}

func TestFirstLine_multiLine(t *testing.T) {
	got := firstLine("First\nSecond\nThird")
	if got != "First" {
		t.Errorf("got %q, want First", got)
	}
}

func TestFirstLine_empty(t *testing.T) {
	got := firstLine("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFirstLine_leadingNewline(t *testing.T) {
	got := firstLine("\nSecond")
	if got != "" {
		t.Errorf("got %q, want empty string (newline is first char)", got)
	}
}

// --- memoTitle ---

func TestMemoTitle_noHeader(t *testing.T) {
	got := memoTitle("Plain text")
	if got != "Plain text" {
		t.Errorf("got %q, want plain text unchanged", got)
	}
}

func TestMemoTitle_h1(t *testing.T) {
	got := memoTitle("# My Memo")
	if got != "My Memo" {
		t.Errorf("got %q, want %q", got, "My Memo")
	}
}

func TestMemoTitle_h2(t *testing.T) {
	got := memoTitle("## Section")
	if got != "Section" {
		t.Errorf("got %q, want %q", got, "Section")
	}
}

func TestMemoTitle_h6(t *testing.T) {
	got := memoTitle("###### Deep")
	if got != "Deep" {
		t.Errorf("got %q, want %q", got, "Deep")
	}
}

func TestMemoTitle_h7_notStripped(t *testing.T) {
	// 7 hashes is not a valid markdown header
	got := memoTitle("####### Too many")
	if got != "####### Too many" {
		t.Errorf("got %q, want original unchanged", got)
	}
}

func TestMemoTitle_hashtagNotStripped(t *testing.T) {
	// no space after # → it's a hashtag, not a header
	got := memoTitle("#tag content here")
	if got != "#tag content here" {
		t.Errorf("got %q, want hashtag unchanged", got)
	}
}

func TestMemoTitle_multipleHashtagsNotStripped(t *testing.T) {
	got := memoTitle("#foo #bar")
	if got != "#foo #bar" {
		t.Errorf("got %q, want hashtags unchanged", got)
	}
}

func TestMemoTitle_headerTrimmed(t *testing.T) {
	got := memoTitle("#   Extra spaces")
	if got != "Extra spaces" {
		t.Errorf("got %q, want leading spaces trimmed", got)
	}
}

func TestMemoTitle_emptyHeader(t *testing.T) {
	// "# " with no content → empty string
	got := memoTitle("# ")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestMemoTitle_usesFirstLine(t *testing.T) {
	got := memoTitle("# Title\nBody text here")
	if got != "Title" {
		t.Errorf("got %q, want only first line stripped", got)
	}
}

func TestMemoTitle_bodyNotStripped(t *testing.T) {
	got := memoTitle("First line\n# Not a title")
	if got != "First line" {
		t.Errorf("got %q, want first line returned", got)
	}
}

// --- truncate ---

func TestTruncate_short(t *testing.T) {
	got := truncate("Hi", 10)
	if got != "Hi" {
		t.Errorf("got %q, want Hi (no truncation)", got)
	}
}

func TestTruncate_exactMax(t *testing.T) {
	got := truncate("12345", 5)
	if got != "12345" {
		t.Errorf("got %q, want 12345", got)
	}
}

func TestTruncate_over(t *testing.T) {
	got := truncate("Hello World", 8)
	// 7 runes + ellipsis
	if got != "Hello W…" {
		t.Errorf("got %q, want Hello W…", got)
	}
}

func TestTruncate_zeroMax(t *testing.T) {
	got := truncate("anything", 0)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestTruncate_unicode(t *testing.T) {
	// "日本語" = 3 runes
	got := truncate("日本語テスト", 4)
	if got != "日本語…" {
		t.Errorf("got %q, want 日本語…", got)
	}
}

// --- tagFilter ---

func TestTagFilter_empty(t *testing.T) {
	got := tagFilter("")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestTagFilter_withTag(t *testing.T) {
	got := tagFilter("work")
	want := `content.contains("#work")`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTagFilter_hierarchical(t *testing.T) {
	got := tagFilter("tracing/jaeger")
	want := `content.contains("#tracing/jaeger")`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- rootTagsFrom ---

func TestRootTagsFrom_flat(t *testing.T) {
	tags := []string{"shopping", "work", "ideas"}
	got := rootTagsFrom(tags)
	want := []string{"ideas", "shopping", "work"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestRootTagsFrom_hierarchical(t *testing.T) {
	tags := []string{"shopping", "tracing/jaeger", "tracing/prometheus", "work"}
	got := rootTagsFrom(tags)
	want := []string{"shopping", "tracing", "work"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestRootTagsFrom_deduplicatesRoots(t *testing.T) {
	tags := []string{"tracing/jaeger", "tracing/prometheus", "tracing"}
	got := rootTagsFrom(tags)
	if len(got) != 1 || got[0] != "tracing" {
		t.Errorf("got %v, want [tracing]", got)
	}
}

func TestRootTagsFrom_empty(t *testing.T) {
	got := rootTagsFrom(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// --- hasSubtags ---

func TestHasSubtags_true(t *testing.T) {
	tags := []string{"shopping", "tracing/jaeger", "work"}
	if !hasSubtags(tags, "tracing") {
		t.Error("expected true for tracing with subtags")
	}
}

func TestHasSubtags_false_flatTag(t *testing.T) {
	tags := []string{"shopping", "work"}
	if hasSubtags(tags, "work") {
		t.Error("expected false for work with no subtags")
	}
}

func TestHasSubtags_false_noSuchRoot(t *testing.T) {
	tags := []string{"work", "tracing/jaeger"}
	if hasSubtags(tags, "shopping") {
		t.Error("expected false for absent root")
	}
}

func TestHasSubtags_doesNotMatchPrefix(t *testing.T) {
	// "tracingmore" should not match root "tracing"
	tags := []string{"tracingmore"}
	if hasSubtags(tags, "tracing") {
		t.Error("expected false: tracingmore is not a subtag of tracing")
	}
}

// --- combineFilters ---

func TestCombineFilters_empty(t *testing.T) {
	got := combineFilters("", "")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestCombineFilters_oneFilter(t *testing.T) {
	got := combineFilters("", `content.contains("#work")`, "")
	want := `content.contains("#work")`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCombineFilters_twoFilters(t *testing.T) {
	got := combineFilters(`content.contains("#work")`, `content.contains("#golang")`)
	want := `content.contains("#work") && content.contains("#golang")`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCombineFilters_skipsEmpty(t *testing.T) {
	got := combineFilters("", `content.contains("#work")`)
	want := `content.contains("#work")`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- sortPinnedFirst ---

func TestSortPinnedFirst_empty(t *testing.T) {
	got := sortPinnedFirst(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestSortPinnedFirst_allPinned(t *testing.T) {
	memos := []model.Memo{
		{Name: "memos/1", Pinned: true},
		{Name: "memos/2", Pinned: true},
	}
	got := sortPinnedFirst(memos)
	if len(got) != 2 || got[0].Name != "memos/1" || got[1].Name != "memos/2" {
		t.Errorf("got %v", got)
	}
}

func TestSortPinnedFirst_nonePinned(t *testing.T) {
	memos := []model.Memo{
		{Name: "memos/1"},
		{Name: "memos/2"},
	}
	got := sortPinnedFirst(memos)
	if len(got) != 2 || got[0].Name != "memos/1" || got[1].Name != "memos/2" {
		t.Errorf("got %v", got)
	}
}

func TestSortPinnedFirst_mixed(t *testing.T) {
	memos := []model.Memo{
		{Name: "memos/1", Pinned: false},
		{Name: "memos/2", Pinned: true},
		{Name: "memos/3", Pinned: false},
		{Name: "memos/4", Pinned: true},
	}
	got := sortPinnedFirst(memos)
	if len(got) != 4 {
		t.Fatalf("expected 4, got %d", len(got))
	}
	// Pinned first, then unpinned
	if !got[0].Pinned || !got[1].Pinned {
		t.Errorf("first two should be pinned, got %v %v", got[0].Name, got[1].Name)
	}
	if got[2].Pinned || got[3].Pinned {
		t.Errorf("last two should be unpinned, got %v %v", got[2].Name, got[3].Name)
	}
}

func TestSortPinnedFirst_stableOrder(t *testing.T) {
	memos := []model.Memo{
		{Name: "memos/A", Pinned: false},
		{Name: "memos/B", Pinned: true},
		{Name: "memos/C", Pinned: false},
		{Name: "memos/D", Pinned: true},
	}
	got := sortPinnedFirst(memos)
	// Pinned: B then D (original order)
	if got[0].Name != "memos/B" || got[1].Name != "memos/D" {
		t.Errorf("pinned order wrong: %v %v", got[0].Name, got[1].Name)
	}
	// Unpinned: A then C (original order)
	if got[2].Name != "memos/A" || got[3].Name != "memos/C" {
		t.Errorf("unpinned order wrong: %v %v", got[2].Name, got[3].Name)
	}
}

// --- listModel.visibleHeight ---

func TestVisibleHeight_withTags(t *testing.T) {
	m := listModel{
		height: 20,
		tags:   []string{"a", "b"},
	}
	// tags no longer subtract from height (shown in popup instead)
	// height(20) - 3 (base overhead) = 17
	got := m.visibleHeight()
	if got != 17 {
		t.Errorf("visibleHeight = %d, want 17", got)
	}
}

func TestVisibleHeight_noTags(t *testing.T) {
	m := listModel{height: 20}
	// height(20) - 3 (base overhead) = 17
	got := m.visibleHeight()
	if got != 17 {
		t.Errorf("visibleHeight = %d, want 17", got)
	}
}

func TestVisibleHeight_minimum(t *testing.T) {
	m := listModel{height: 1, tags: []string{"a"}}
	got := m.visibleHeight()
	if got < 1 {
		t.Errorf("visibleHeight = %d, must be ≥ 1", got)
	}
}

// --- adjustOffset ---

func TestAdjustOffset_scrollDown(t *testing.T) {
	a := &App{
		list: listModel{
			memos:  make([]model.Memo, 20),
			cursor: 15,
			offset: 0,
			height: 14, // visibleHeight ≈ 11
			tags:   nil,
		},
	}
	a.adjustOffset()
	if a.list.offset > a.list.cursor {
		t.Errorf("offset %d > cursor %d", a.list.offset, a.list.cursor)
	}
	visible := a.list.visibleHeight()
	if a.list.cursor >= a.list.offset+visible {
		t.Errorf("cursor %d not visible (offset=%d, visible=%d)", a.list.cursor, a.list.offset, visible)
	}
}

func TestAdjustOffset_scrollUp(t *testing.T) {
	a := &App{
		list: listModel{
			memos:  make([]model.Memo, 20),
			cursor: 2,
			offset: 10,
			height: 14,
		},
	}
	a.adjustOffset()
	if a.list.offset != 2 {
		t.Errorf("offset = %d, want 2 (cursor brought offset down)", a.list.offset)
	}
}

func TestAdjustOffset_noop(t *testing.T) {
	a := &App{
		list: listModel{
			memos:  make([]model.Memo, 20),
			cursor: 3,
			offset: 0,
			height: 14,
		},
	}
	a.adjustOffset()
	if a.list.offset != 0 {
		t.Errorf("offset should stay 0, got %d", a.list.offset)
	}
}

// --- listModel.selected ---

func TestSelected_empty(t *testing.T) {
	m := listModel{}
	if m.selected() != nil {
		t.Error("expected nil for empty list")
	}
}

func TestSelected_valid(t *testing.T) {
	m := listModel{
		memos:  []model.Memo{{Name: "memos/1"}, {Name: "memos/2"}},
		cursor: 1,
	}
	sel := m.selected()
	if sel == nil || sel.Name != "memos/2" {
		t.Errorf("selected = %v, want memos/2", sel)
	}
}

func TestSelected_outOfBounds(t *testing.T) {
	m := listModel{
		memos:  []model.Memo{{Name: "memos/1"}},
		cursor: 5,
	}
	if m.selected() != nil {
		t.Error("expected nil when cursor out of bounds")
	}
}
