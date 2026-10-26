package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/warrensbox/whosup/internal/ports"
)

var (
	borderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cellStyle     = lipgloss.NewStyle().Padding(0, 1)
	headerStyle   = cellStyle.Bold(true)
	selectedStyle = cellStyle.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("#2277cc"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dialogStyle   = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2)
	labelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true).Align(lipgloss.Right)
	buttonStyle       = lipgloss.NewStyle().Padding(0, 2).Background(lipgloss.Color("238"))
	activeButtonStyle = buttonStyle.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("#2277cc"))
)

// After a stop signal, check this often whether the process has exited.
const (
	pollInterval = 250 * time.Millisecond
	stopTries    = 20 // 5s for a graceful stop
	killTries    = 8  // 2s for a force kill
)

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type dialogKind int

const (
	noDialog       dialogKind = iota
	stopDialog                // "Are you sure you want to stop this service?"
	progressDialog            // spinner while waiting for the process to exit
	killDialog                // "Still running. Force kill it?"
	resultDialog              // final outcome, closed with enter
	detailsDialog             // every field of a row, including the full path
)

// Messages sent back into Update by commands.
type (
	signalMsg struct {
		port   ports.Port
		forced bool
		err    error
	}
	checkMsg struct {
		port   ports.Port
		forced bool
		tries  int
	}
	refreshMsg struct{}
	listMsg    struct {
		list []ports.Port
		err  error
	}
)

type model struct {
	list   []ports.Port
	cursor int // selected row in list
	offset int // first row shown
	height int // number of rows that fit on screen

	dialog dialogKind
	yes    bool       // which button is highlighted in the dialog
	target ports.Port // process the dialog is about
	forced bool       // progress dialog is waiting on a force kill
	tries  int        // progress dialog poll count, drives the spinner
	result string     // text of the result dialog
	isErr  bool       // result (or status) is an error
	status string     // line under the table, e.g. reload errors

	// Swappable for tests.
	stop  func(pid int) error
	kill  func(pid int) error
	alive func(pid int) bool
	load  func() ([]ports.Port, error)
	poll  time.Duration
}

func (m model) Init() tea.Cmd { return nil }

// Key names used in more than one place.
const (
	keyEnter = "enter"
	keyEsc   = "esc"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.dialog != noDialog {
			return m.updateDialog(msg)
		}
		return m.updateTable(msg)
	case tea.WindowSizeMsg:
		// Each row takes two lines (row + separator). Leave room for the
		// top/bottom border, header, header separator, status and help line.
		m.height = max((msg.Height-5)/2, 1)
		m.moveCursor(0)
	case signalMsg:
		return m.onSignal(msg)
	case checkMsg:
		return m.onCheck(msg)
	case refreshMsg:
		load := m.load
		return m, func() tea.Msg {
			list, err := load()
			return listMsg{list, err}
		}
	case listMsg:
		if msg.err != nil {
			m.status = "Could not reload ports: " + msg.err.Error()
			return m, nil
		}
		m.status = ""
		m.setList(msg.list)
		return m, nil
	}
	return m, nil
}

// updateTable handles keys while no dialog is open.
func (m model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyEsc, "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case keyEnter:
		if len(m.list) > 0 {
			m.dialog, m.yes, m.target = stopDialog, false, m.list[m.cursor] // default to No
		}
	case "m":
		if len(m.list) > 0 {
			m.dialog, m.target = detailsDialog, m.list[m.cursor]
		}
	case "r":
		return m, refresh
	}
	return m, nil
}

// onSignal handles the result of sending SIGTERM or SIGKILL.
func (m model) onSignal(msg signalMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		verb := "stop"
		if msg.forced {
			verb = "force kill"
		}
		m.showResult(true, "Could not %s %s:\n%v", verb, name(msg.port), msg.err)
		return m, nil
	}
	return m, m.check(msg.port, msg.forced, 1)
}

// onCheck handles a poll of whether the signalled process has exited.
func (m model) onCheck(msg checkMsg) (tea.Model, tea.Cmd) {
	m.tries = msg.tries
	if !m.alive(msg.port.PID) {
		verb := "Stopped"
		if msg.forced {
			verb = "Force killed"
		}
		m.showResult(false, "✓ %s %s", verb, name(msg.port))
		return m, refresh
	}
	limit := stopTries
	if msg.forced {
		limit = killTries
	}
	if msg.tries < limit {
		return m, m.check(msg.port, msg.forced, msg.tries+1)
	}
	if !msg.forced {
		m.dialog, m.yes = killDialog, false
		return m, nil
	}
	m.showResult(true, "%s is still running\nafter a force kill.", name(msg.port))
	return m, refresh
}

// moveCursor moves the selection by delta and scrolls to keep it visible.
func (m *model) moveCursor(delta int) {
	m.cursor = max(min(m.cursor+delta, len(m.list)-1), 0)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
	m.offset = max(min(m.offset, len(m.list)-m.height), 0)
}

func (m model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.dialog {
	case progressDialog:
		return m, nil // wait for the outcome
	case resultDialog, detailsDialog:
		switch msg.String() {
		case keyEnter, keyEsc, "q", " ", "m":
			m.dialog = noDialog
		}
		return m, nil
	}
	switch msg.String() {
	case "left", "right", "h", "l", "tab", "shift+tab":
		m.yes = !m.yes
	case "y":
		m.yes = true
		return m.confirm()
	case "n", keyEsc, "q":
		m.yes = false
		return m.confirm()
	case keyEnter:
		return m.confirm()
	}
	return m, nil
}

func (m model) confirm() (tea.Model, tea.Cmd) {
	p := m.target
	switch {
	case m.dialog == stopDialog && m.yes:
		m.dialog, m.forced, m.tries = progressDialog, false, 0
		stop := m.stop
		return m, func() tea.Msg { return signalMsg{port: p, err: stop(p.PID)} }
	case m.dialog == killDialog && m.yes:
		m.dialog, m.forced, m.tries = progressDialog, true, 0
		kill := m.kill
		return m, func() tea.Msg { return signalMsg{port: p, forced: true, err: kill(p.PID)} }
	case m.dialog == killDialog:
		m.showResult(true, "%s is still running.", name(p))
		return m, refresh
	}
	m.dialog = noDialog
	return m, nil
}

// check waits one poll interval, then asks whether p has exited.
func (m model) check(p ports.Port, forced bool, tries int) tea.Cmd {
	return tea.Tick(m.poll, func(time.Time) tea.Msg {
		return checkMsg{port: p, forced: forced, tries: tries}
	})
}

func refresh() tea.Msg { return refreshMsg{} }

func (m *model) showResult(isErr bool, format string, args ...any) {
	m.dialog, m.result, m.isErr = resultDialog, fmt.Sprintf(format, args...), isErr
}

func name(p ports.Port) string { return fmt.Sprintf("%s (PID %d)", p.Command, p.PID) }

func (m *model) setList(list []ports.Port) {
	m.list = list
	m.moveCursor(0)
}

func (m model) View() string {
	out := m.tableView()
	if m.dialog != noDialog {
		out = overlay(out, m.dialogView())
	}
	out += "\n"
	if m.status != "" {
		out += errorStyle.Render(m.status) + "\n"
	}
	return out + helpStyle.Render(m.help()) + "\n"
}

func (m model) help() string {
	switch m.dialog {
	case stopDialog, killDialog:
		return "←/→ choose • enter confirm • y yes • n/esc no"
	case progressDialog:
		return "waiting for the process to exit… • ctrl+c quit"
	case resultDialog, detailsDialog:
		return "enter/esc close"
	}
	return "↑/k up • ↓/j down • m more • r refresh • enter stop service • q quit"
}

func (m model) dialogView() string {
	p := m.target
	var body string
	switch m.dialog {
	case stopDialog:
		body = fmt.Sprintf("Are you sure you want to stop this service?\n\n%s on port %d", name(p), p.Port)
	case killDialog:
		body = fmt.Sprintf("%s is still running.\n\nForce kill it? It won't get a chance to clean up.", name(p))
	case progressDialog:
		verb, limit := "Stopping", stopTries
		if m.forced {
			verb, limit = "Force killing", killTries
		}
		left := time.Duration(limit-m.tries) * pollInterval
		body = fmt.Sprintf("%s %s %s\n\nWaiting up to %.0fs for it to exit",
			spinner[m.tries%len(spinner)], verb, name(p), left.Seconds())
	case detailsDialog:
		return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, detailsView(p), "", activeButtonStyle.Render("Close")))
	case resultDialog:
		style := successStyle
		if m.isErr {
			style = errorStyle
		}
		body = style.Render(m.result)
		return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, body, "", activeButtonStyle.Render("OK")))
	}
	if m.dialog == progressDialog {
		return dialogStyle.Render(body)
	}
	yes, no := buttonStyle.Render("Yes"), activeButtonStyle.Render("No")
	if m.yes {
		yes, no = activeButtonStyle.Render("Yes"), buttonStyle.Render("No")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yes, "  ", no)
	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, body, "", buttons))
}

// detailsValueWidth wraps long values, like deep paths, in the details dialog.
const detailsValueWidth = 60

// detailsFields lists every field of a row as label/value pairs.
func detailsFields(p ports.Port) [][2]string {
	return [][2]string{
		{"Port", p.Label()},
		{"PID", strconv.Itoa(p.PID)},
		{"Command", p.Command},
		{"Address", describeAddress(p.Address)},
		{"CPU", p.CPU},
		{"Memory", p.Memory},
		{"Directory", p.Dir},
	}
}

// labelWidth is the width of the longest label in fields.
func labelWidth(fields [][2]string) int {
	w := 0
	for _, f := range fields {
		w = max(w, len(f[0]))
	}
	return w
}

// Details renders every field of a row as plain, unstyled "label  value" lines.
func Details(p ports.Port) string {
	fields := detailsFields(p)
	labelW := labelWidth(fields)
	var b strings.Builder
	for _, f := range fields {
		fmt.Fprintf(&b, "%-*s  %s\n", labelW, f[0], f[1])
	}
	return b.String()
}

func detailsView(p ports.Port) string {
	fields := detailsFields(p)
	labelW := labelWidth(fields)
	valueW := 0
	for _, f := range fields {
		valueW = max(valueW, lipgloss.Width(f[1]))
	}
	valueStyle := lipgloss.NewStyle().Width(min(valueW, detailsValueWidth))
	lines := make([]string, 0, len(fields))
	for _, f := range fields {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Width(labelW).Render(f[0]), "  ", valueStyle.Render(f[1])))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// describeAddress explains who can reach a listening address.
func describeAddress(addr string) string {
	switch addr {
	case "*":
		return "* (all interfaces, reachable from the network)"
	case "127.0.0.1", "[::1]", "localhost":
		return addr + " (local only)"
	}
	return addr
}

// numeric columns are right-aligned.
var numeric = map[int]bool{0: true, 1: true, 4: true, 5: true}

func (m model) tableView() string {
	end := min(m.offset+m.height, len(m.list))
	rows := make([][]string, 0, end-m.offset)
	for _, p := range m.list[m.offset:end] {
		rows = append(rows, []string{p.Label(), strconv.Itoa(p.PID), p.Command, p.Address, p.CPU, p.Memory, lastSegments(p.Dir, 2)})
	}
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		BorderRow(true).
		Headers("PORT", "PID", "COMMAND", "ADDRESS", "CPU", "MEMORY", "DIRECTORY").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := cellStyle
			switch {
			case row == table.HeaderRow:
				s = headerStyle
			case m.offset+row == m.cursor:
				s = selectedStyle
			}
			if numeric[col] {
				s = s.Align(lipgloss.Right)
			}
			return s
		})
	return t.Render()
}

// lastSegments keeps the last n parts of a path, the most telling bit,
// e.g. "~/Library/Application Support/Postgres/var-17" -> "Postgres/var-17".
func lastSegments(path string, n int) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) <= n {
		return path
	}
	return strings.Join(parts[len(parts)-n:], "/")
}

func newModel(list []ports.Port, load func() ([]ports.Port, error)) model {
	return model{
		list:   list,
		height: min(len(list), 10),
		stop:   ports.Stop,
		kill:   ports.Kill,
		alive:  ports.Alive,
		load:   load,
		poll:   pollInterval,
	}
}

// Run shows the ports in an interactive, scrollable table. load is called
// to refresh the list.
func Run(list []ports.Port, load func() ([]ports.Port, error)) error {
	_, err := tea.NewProgram(newModel(list, load)).Run()
	return err
}
