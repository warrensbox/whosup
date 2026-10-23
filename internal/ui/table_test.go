package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/warrensbox/whosup/internal/ports"
)

var sample = []ports.Port{
	{Port: 3000, PID: 100, Command: "node"},
	{Port: 5432, PID: 200, Command: "postgres"},
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// send applies msgs in order, running any returned command once.
func send(m model, msgs ...tea.Msg) (model, []tea.Msg) {
	var results []tea.Msg
	for _, msg := range msgs {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd != nil {
			results = append(results, cmd())
		}
	}
	return m, results
}

// fakeProc records signals and decides when the process "exits".
type fakeProc struct {
	stopped, killed int
	stopErr         error
	ignoreTerm      bool // keep running after SIGTERM
	ignoreKill      bool // keep running even after SIGKILL
}

func testModel(f *fakeProc) model {
	m := newModel(sample, ports.List)
	m.poll = time.Millisecond
	m.stop = func(pid int) error { f.stopped = pid; return f.stopErr }
	m.kill = func(pid int) error { f.killed = pid; return nil }
	m.alive = func(_ int) bool {
		if f.killed != 0 {
			return f.ignoreKill
		}
		return f.ignoreTerm
	}
	m.load = func() ([]ports.Port, error) { return sample, nil }
	return m
}

// run feeds msg to the model and keeps feeding the results of returned
// commands until there are none, like the Bubble Tea runtime would.
func run(m model, msg tea.Msg) model {
	for range 100 {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd == nil {
			return m
		}
		msg = cmd()
	}
	panic("too many steps")
}

func runKeys(m model, keys ...string) model {
	for _, k := range keys {
		m = run(m, key(k))
	}
	return m
}

func TestEnterOpensDialogDefaultNo(t *testing.T) {
	var f fakeProc
	m := runKeys(testModel(&f), "enter")
	if m.dialog != stopDialog || m.yes {
		t.Fatalf("dialog=%v yes=%v, want stop/false", m.dialog, m.yes)
	}
	if !strings.Contains(m.View(), "Are you sure you want to stop this service?") {
		t.Error("dialog text missing from view")
	}
	// Enter on the default (No) does nothing.
	m = runKeys(m, "enter")
	if m.dialog != noDialog || f.stopped != 0 {
		t.Fatalf("dialog=%v stopped=%d, want none/0", m.dialog, f.stopped)
	}
}

func TestEscCancels(t *testing.T) {
	var f fakeProc
	m := runKeys(testModel(&f), "enter", "esc")
	if m.dialog != noDialog || f.stopped != 0 {
		t.Fatalf("dialog=%v stopped=%d", m.dialog, f.stopped)
	}
}

func TestGracefulStop(t *testing.T) {
	var f fakeProc
	m := runKeys(testModel(&f), "down", "enter", "left", "enter")
	if f.stopped != 200 || f.killed != 0 {
		t.Fatalf("stopped=%d killed=%d, want 200/0", f.stopped, f.killed)
	}
	if m.dialog != resultDialog || m.result != "✓ Stopped postgres (PID 200)" || m.isErr {
		t.Errorf("dialog=%v result=%q isErr=%v", m.dialog, m.result, m.isErr)
	}
	m = runKeys(m, "enter")
	if m.dialog != noDialog {
		t.Error("enter should close the result dialog")
	}
}

func TestProgressShownInDialog(t *testing.T) {
	var f fakeProc
	m := testModel(&f)
	m = runKeys(m, "enter")
	// Choose Yes without running the stop command, to inspect the progress state.
	next, _ := m.Update(key("y"))
	m = next.(model)
	if m.dialog != progressDialog {
		t.Fatalf("dialog=%v, want progress", m.dialog)
	}
	if !strings.Contains(m.View(), "Stopping node (PID 100)") {
		t.Error("progress text missing from dialog")
	}
	// Keys other than ctrl+c are ignored while waiting.
	next, _ = m.Update(key("esc"))
	if next.(model).dialog != progressDialog {
		t.Error("esc should not close the progress dialog")
	}
}

func TestYShortcut(t *testing.T) {
	var f fakeProc
	runKeys(testModel(&f), "enter", "y")
	if f.stopped != 100 {
		t.Fatalf("stopped PID %d, want 100", f.stopped)
	}
}

func TestSignalErrorShown(t *testing.T) {
	f := fakeProc{stopErr: errors.New("operation not permitted")}
	m := runKeys(testModel(&f), "enter", "y")
	if m.dialog != resultDialog || !m.isErr || !strings.Contains(m.result, "Could not stop node (PID 100):\noperation not permitted") {
		t.Errorf("dialog=%v result=%q isErr=%v", m.dialog, m.result, m.isErr)
	}
}

func TestStillRunningOffersForceKill(t *testing.T) {
	f := fakeProc{ignoreTerm: true}
	m := runKeys(testModel(&f), "enter", "y")
	if m.dialog != killDialog || m.yes {
		t.Fatalf("dialog=%v yes=%v, want kill/false", m.dialog, m.yes)
	}
	if !strings.Contains(m.View(), "node (PID 100) is still running.") {
		t.Error("kill dialog text missing")
	}
	m = runKeys(m, "y")
	if f.killed != 100 {
		t.Fatalf("killed=%d, want 100", f.killed)
	}
	if m.result != "✓ Force killed node (PID 100)" || m.isErr {
		t.Errorf("result=%q isErr=%v", m.result, m.isErr)
	}
}

func TestDeclineForceKill(t *testing.T) {
	f := fakeProc{ignoreTerm: true}
	m := runKeys(testModel(&f), "enter", "y", "n")
	if f.killed != 0 {
		t.Fatal("should not force kill after No")
	}
	if m.result != "node (PID 100) is still running." || !m.isErr {
		t.Errorf("result=%q isErr=%v", m.result, m.isErr)
	}
}

func TestSurvivesForceKill(t *testing.T) {
	f := fakeProc{ignoreTerm: true, ignoreKill: true}
	m := runKeys(testModel(&f), "enter", "y", "y")
	if !strings.Contains(m.result, "still running\nafter a force kill") || !m.isErr {
		t.Errorf("result=%q isErr=%v", m.result, m.isErr)
	}
}

func TestScrollKeepsCursorVisible(t *testing.T) {
	var list []ports.Port
	for i := range 10 {
		list = append(list, ports.Port{Port: 1000 + i, PID: 100 + i, Command: "svc"})
	}
	m := newModel(list, ports.List)
	m, _ = send(m, tea.WindowSizeMsg{Height: 11}) // room for 3 rows
	if m.height != 3 {
		t.Fatalf("height = %d, want 3", m.height)
	}
	for range 5 {
		m, _ = send(m, key("down"))
	}
	if m.cursor != 5 || m.offset != 3 {
		t.Fatalf("cursor=%d offset=%d, want 5/3", m.cursor, m.offset)
	}
	if !strings.Contains(m.View(), "1005") || strings.Contains(m.View(), "1002") {
		t.Error("view should show rows 3–5 only")
	}
	for range 20 {
		m, _ = send(m, key("up"))
	}
	if m.cursor != 0 || m.offset != 0 {
		t.Fatalf("cursor=%d offset=%d, want 0/0", m.cursor, m.offset)
	}
}

func TestLastSegments(t *testing.T) {
	cases := map[string]string{
		"~/Library/Application Support/Postgres/var-17": "Postgres/var-17",
		"~/workspace/interview-prep":                    "workspace/interview-prep",
		"~/app":                                         "~/app",
		"~":                                             "~",
		"/":                                             "/",
		"-":                                             "-",
	}
	for in, want := range cases {
		if got := lastSegments(in, 2); got != want {
			t.Errorf("lastSegments(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMoreShowsDetails(t *testing.T) {
	list := []ports.Port{{Port: 5432, PID: 200, Command: "postgres", Address: "127.0.0.1",
		CPU: "0.0%", Memory: "2.2 MB", Dir: "~/Library/Application Support/Postgres/var-17"}}
	m, _ := send(newModel(list, ports.List), tea.WindowSizeMsg{Height: 30})
	view := m.View()
	if !strings.Contains(view, "Postgres/var-17") || strings.Contains(view, "Application Support") {
		t.Error("table should show only the last two path segments")
	}
	if strings.Contains(view, "[more]") {
		t.Error("table should not have a [more] column")
	}
	m = runKeys(m, "m")
	if m.dialog != detailsDialog {
		t.Fatalf("dialog=%v, want details", m.dialog)
	}
	view = m.View()
	for _, want := range []string{"~/Library/Application Support/Postgres/var-17", "127.0.0.1 (local only)", "2.2 MB", "postgres"} {
		if !strings.Contains(view, want) {
			t.Errorf("details missing %q", want)
		}
	}
	if m = runKeys(m, "esc"); m.dialog != noDialog {
		t.Error("esc should close details")
	}
}

func TestRefreshReloadsList(t *testing.T) {
	var f fakeProc
	m := testModel(&f)
	updated := []ports.Port{{Port: 8080, PID: 300, Command: "go"}}
	m.load = func() ([]ports.Port, error) { return updated, nil }
	m = runKeys(m, "r")
	if len(m.list) != 1 || m.list[0].Port != 8080 {
		t.Fatalf("list = %+v, want the reloaded list", m.list)
	}
}

func TestDetails(t *testing.T) {
	p := ports.Port{Port: 3000, PID: 42, Command: "node", Address: "127.0.0.1", CPU: "1.5%", Memory: "10.0 MB", Dir: "~/app"}
	want := `Port       3000
PID        42
Command    node
Address    127.0.0.1 (local only)
CPU        1.5%
Memory     10.0 MB
Directory  ~/app
`
	if got := Details(p); got != want {
		t.Errorf("Details() =\n%s\nwant:\n%s", got, want)
	}
}
