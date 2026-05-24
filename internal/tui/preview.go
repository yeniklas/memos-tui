package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/yeniklas/memos-tui/internal/config"
	"github.com/yeniklas/memos-tui/internal/model"
)

type previewModel struct {
	vp              viewport.Model
	memo            *model.Memo
	width           int
	height          int
	ready           bool
	markdownEnabled bool
	theme           config.Theme
	renderer        *glamour.TermRenderer
}

func newPreviewModel(markdownEnabled bool, theme config.Theme) previewModel {
	return previewModel{
		markdownEnabled: markdownEnabled,
		theme:           theme,
	}
}

func (m *previewModel) setSize(w, h int) {
	m.width = w
	m.height = h
	innerW := w - 4
	innerH := h - 4
	if innerH < 1 {
		innerH = 1
	}
	if !m.ready {
		m.vp = viewport.New(innerW, innerH)
		m.ready = true
	} else {
		m.vp.Width = innerW
		m.vp.Height = innerH
	}
	if m.markdownEnabled {
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithStyles(buildGlamourStyle(m.theme)),
			glamour.WithWordWrap(innerW),
		)
	}
	m.refresh()
}

func (m *previewModel) setMemo(memo *model.Memo) {
	m.memo = memo
	m.refresh()
}

func (m *previewModel) refresh() {
	if m.memo == nil || !m.ready {
		m.vp.SetContent(styleMuted.Render("(select a memo)"))
		return
	}
	content := m.memo.Content
	if m.markdownEnabled && m.renderer != nil {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = strings.TrimRight(rendered, "\n")
		}
	}
	m.vp.SetContent(content)
	m.vp.GotoTop()
}

func (m previewModel) View(focused bool) string {
	style := stylePanel
	if focused {
		style = stylePanelFocused
	}

	inner := m.renderInner()
	return style.Width(m.width - 2).Height(m.height - 2).Render(inner)
}

func (m previewModel) renderInner() string {
	if m.memo == nil {
		return styleMuted.Render("(select a memo to preview)")
	}

	meta := m.renderMeta()
	sep := styleMuted.Render(strings.Repeat("─", max(m.width-4, 0)))

	var parts []string
	parts = append(parts, meta)
	parts = append(parts, sep)
	if m.ready {
		parts = append(parts, m.vp.View())
	}
	return strings.Join(parts, "\n")
}

func (m previewModel) renderMeta() string {
	if m.memo == nil {
		return ""
	}
	date := m.memo.DisplayTime.Format("2006-01-02 15:04")
	vis := m.memo.Visibility.Short()
	tags := ""
	if len(m.memo.Tags) > 0 {
		tagged := make([]string, len(m.memo.Tags))
		for i, t := range m.memo.Tags {
			tagged[i] = "#" + t
		}
		tags = "  " + styleTag.Render(strings.Join(tagged, " "))
	}
	return styleMetaBar.Render(fmt.Sprintf("%s · %s%s", date, vis, tags))
}

func buildGlamourStyle(t config.Theme) ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			Margin: uintPtr(0),
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:       strPtr(t.Primary),
				Bold:        boolPtr(true),
				BlockSuffix: "\n",
			},
		},
		H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Bold: boolPtr(true), Prefix: "# ",
		}},
		H2: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Bold: boolPtr(true), Prefix: "## ",
		}},
		H3: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Prefix: "### ",
		}},
		H4: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Prefix: "#### ",
		}},
		H5: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Prefix: "##### ",
		}},
		H6: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Color: strPtr(t.Primary), Prefix: "###### ",
		}},
		Strikethrough: ansi.StylePrimitive{CrossedOut: boolPtr(true)},
		Strong:        ansi.StylePrimitive{Bold: boolPtr(true)},
		Emph:          ansi.StylePrimitive{Italic: boolPtr(true)},
		HorizontalRule: ansi.StylePrimitive{
			Color:  strPtr(t.Muted),
			Format: "\n--------\n",
		},
		// Lists — structural settings required for bullet/number rendering.
		List: ansi.StyleList{LevelIndent: 2},
		Item: ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{BlockPrefix: ". "},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{},
			Ticked:         "[✓] ",
			Unticked:       "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:     strPtr(t.Tag),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: strPtr(t.Primary),
			Bold:  boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:     strPtr(t.Muted),
			Underline: boolPtr(true),
		},
		ImageText: ansi.StylePrimitive{
			Color:  strPtr(t.Muted),
			Format: "Image: {{.text}} →",
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strPtr(t.Muted),
				Italic: boolPtr(true),
			},
			Indent:      uintPtr(1),
			IndentToken: strPtr("│ "),
		},
		// Inline code: colored with tag color, padded for readability.
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  strPtr(t.Tag),
				Prefix: " ",
				Suffix: " ",
			},
		},
		// Fenced code blocks: monokai syntax highlighting, small margin.
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: uintPtr(1),
			},
			Theme: "monokai",
		},
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }
