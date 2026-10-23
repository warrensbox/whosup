package ports

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Port is a listening TCP port and the process that owns it.
type Port struct {
	Port    int
	PID     int
	Command string
	Address string
	CPU     string
	Memory  string
	Dir     string
	User    string // owner of the process
	Exe     string // path to the executable, if known
}

// List returns listening TCP ports using lsof.
func List() ([]Port, error) {
	out, err := exec.Command("lsof", "+c", "0", "-nP", "-iTCP", "-sTCP:LISTEN").Output()
	if err != nil {
		// lsof exits 1 when nothing matches.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && len(out) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("running lsof: %w", err)
	}
	list := parse(out)
	addStats(list)
	addDirs(list)
	return list, nil
}

func parse(out []byte) []Port {
	seen := map[string]bool{}
	var result []Port
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Scan() // skip header
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME (LISTEN)
		if len(f) < 9 {
			continue
		}
		name := f[8]
		i := strings.LastIndex(name, ":")
		if i < 0 {
			continue
		}
		port, err := strconv.Atoi(name[i+1:])
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		key := f[1] + name
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, Port{Port: port, PID: pid, Command: strings.ReplaceAll(f[0], `\x20`, " "), Address: name[:i], User: f[2]})
	}
	sort.Slice(result, func(a, b int) bool { return result[a].Port < result[b].Port })
	return result
}
