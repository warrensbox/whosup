package ports

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// stats is a process's CPU and memory usage, and its executable.
type stats struct {
	cpu   float64 // percent
	rssKB int     // resident memory in KB
	exe   string  // full path on macOS, just the name on Linux
}

// addStats fills in CPU, Memory and Exe for each port using a single ps call.
// If ps fails, the fields are left as "-".
func addStats(list []Port) {
	if len(list) == 0 {
		return
	}
	pids := make([]string, 0, len(list))
	for _, p := range list {
		pids = append(pids, strconv.Itoa(p.PID))
	}
	// ps exits non-zero if any PID has gone, but still prints the rest.
	out, _ := exec.Command("ps", "-o", "pid=,%cpu=,rss=,comm=", "-p", strings.Join(pids, ",")).Output() //nolint:errcheck,gosec // see above; the only variable args are PIDs from strconv.Itoa
	all := parseStats(out)
	for i := range list {
		s, ok := all[list[i].PID]
		if !ok {
			list[i].CPU, list[i].Memory = "-", "-"
			continue
		}
		list[i].CPU = fmt.Sprintf("%.1f%%", s.cpu)
		list[i].Memory = formatKB(s.rssKB)
		list[i].Exe = s.exe
		if !strings.HasPrefix(s.exe, "/") {
			// Linux ps only gives the name; /proc has the path, if we may read it.
			if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", list[i].PID)); err == nil {
				list[i].Exe = exe
			}
		}
	}
}

// parseStats maps PID to its stats from "pid %cpu rss comm" lines.
func parseStats(out []byte) map[int]stats {
	all := map[int]stats{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		cpu, err2 := strconv.ParseFloat(f[1], 64)
		rss, err3 := strconv.Atoi(f[2])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		// comm is last because it can contain spaces.
		all[pid] = stats{cpu: cpu, rssKB: rss, exe: strings.Join(f[3:], " ")}
	}
	return all
}

// formatKB renders a KB count as KB, MB or GB.
func formatKB(kb int) string {
	switch {
	case kb >= 1024*1024:
		return fmt.Sprintf("%.1f GB", float64(kb)/(1024*1024))
	case kb >= 1024:
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	default:
		return fmt.Sprintf("%d KB", kb)
	}
}
