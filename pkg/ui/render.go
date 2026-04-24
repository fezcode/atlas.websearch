package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	sepGlyph     = "·"
	sectionOpen  = "┤"
	sectionClose = "├"
)

// section renders a rounded box with the title inlined into the top border:
//
//	╭── ┤ §NN  TITLE ├ ────────╮
//	│   body                    │
//	╰───────────────────────────╯
func section(num, title, body string, width int) string {
	if width < 20 {
		width = 20
	}
	titleInner := sSectionKey.Render(fmt.Sprintf(" §%s  ", num)) +
		sSectionTitle.Render(title) +
		" "
	label := sBorder.Render(sectionOpen) + titleInner + sBorder.Render(sectionClose)
	labelW := lipgloss.Width(label)

	lead := 2
	fill := width - 2 - lead - labelW
	if fill < 1 {
		lead = 1
		fill = width - 2 - lead - labelW
		if fill < 1 {
			fill = 1
		}
	}
	top := sBorder.Render("╭"+strings.Repeat("─", lead)) +
		label +
		sBorder.Render(strings.Repeat("─", fill)+"╮")

	inner := width - 4
	var rows []string
	for _, ln := range strings.Split(body, "\n") {
		w := lipgloss.Width(ln)
		if w < inner {
			ln = ln + strings.Repeat(" ", inner-w)
		} else if w > inner {
			ln = truncateVisible(ln, inner)
		}
		rows = append(rows,
			sBorder.Render("│")+" "+ln+" "+sBorder.Render("│"))
	}
	bottom := sBorder.Render("╰" + strings.Repeat("─", width-2) + "╯")

	return top + "\n" + strings.Join(rows, "\n") + "\n" + bottom
}

// truncateVisible cuts `s` to visible width `n` (adds ellipsis when clipped).
// Strips lipgloss styling on clip — callers should size correctly when styled.
func truncateVisible(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// labelValue produces "LABEL        value" with a fixed label column.
func labelValue(label, value string, labelCol int) string {
	lbl := sLabel.Render(label)
	pad := labelCol - lipgloss.Width(lbl)
	if pad < 1 {
		pad = 1
	}
	return lbl + strings.Repeat(" ", pad) + value
}

// horiz joins parts with a thin separator pill.
func horiz(parts ...string) string {
	return strings.Join(parts, "  "+sBorder.Render(sepGlyph)+"  ")
}

// padRight right-aligns s within n visible cells.
func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return strings.Repeat(" ", n-w) + s
}

// padLeft left-aligns s within n visible cells.
func padLeft(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func nonempty(s, fb string) string {
	if strings.TrimSpace(s) == "" {
		return fb
	}
	return s
}
