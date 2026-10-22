package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestOverlayCentersOnBackground(t *testing.T) {
	bg := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbbbbbbb",
		"cccccccccc",
	}, "\n")
	got := strings.Split(ansi.Strip(overlay(bg, "XX")), "\n")
	want := []string{"aaaaaaaaaa", "bbbbXXbbbb", "cccccccccc"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOverlayTallerThanBackground(t *testing.T) {
	got := strings.Split(ansi.Strip(overlay("abcd", "1\n2\n3")), "\n")
	if len(got) != 3 {
		t.Fatalf("got %d lines, want 3", len(got))
	}
	if !strings.Contains(got[0], "1") || !strings.Contains(got[2], "3") {
		t.Errorf("fg not drawn: %q", got)
	}
}
