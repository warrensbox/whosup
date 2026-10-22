package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

// overlay draws fg centered on top of bg, with bg dimmed so fg stands out
// like a modal. bg is padded with blank lines if it is shorter than fg.
func overlay(bg, fg string) string {
	bgLines := strings.Split(ansi.Strip(bg), "\n")
	fgLines := strings.Split(fg, "\n")
	bgW, fgW := lipgloss.Width(bg), lipgloss.Width(fg)

	for len(bgLines) < len(fgLines) {
		bgLines = append(bgLines, "")
	}
	x := max((bgW-fgW)/2, 0)
	y := (len(bgLines) - len(fgLines)) / 2

	out := make([]string, len(bgLines))
	for i, line := range bgLines {
		j := i - y
		if j < 0 || j >= len(fgLines) {
			out[i] = dimStyle.Render(line)
			continue
		}
		// Pad short lines so the dialog lands at column x.
		if w := ansi.StringWidth(line); w < x {
			line += strings.Repeat(" ", x-w)
		}
		left := ansi.Truncate(line, x, "")
		right := ansi.TruncateLeft(line, x+fgW, "")
		out[i] = dimStyle.Render(left) + fgLines[j] + dimStyle.Render(right)
	}
	return strings.Join(out, "\n")
}
