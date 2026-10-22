package ports

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// addDirs fills in Dir with each process's working directory using a
// single lsof call. Processes we can't inspect are left as "-".
func addDirs(list []Port) {
	if len(list) == 0 {
		return
	}
	pids := make([]string, 0, len(list))
	for _, p := range list {
		pids = append(pids, strconv.Itoa(p.PID))
	}
	// lsof exits non-zero if any PID can't be read, but still prints the rest.
	out, _ := exec.Command("lsof", "-a", "-p", strings.Join(pids, ","), "-d", "cwd", "-Fn").Output()
	home, _ := os.UserHomeDir()
	dirs := parseDirs(out)
	for i := range list {
		dir, ok := dirs[list[i].PID]
		if !ok {
			list[i].Dir = "-"
			continue
		}
		list[i].Dir = shortenHome(dir, home)
	}
}

// parseDirs reads lsof -Fn output ("p<pid>", "fcwd", "n<path>" lines).
func parseDirs(out []byte) map[int]string {
	dirs := map[int]string{}
	pid := 0
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
		case 'n':
			if pid != 0 {
				dirs[pid] = line[1:]
			}
		}
	}
	return dirs
}

// shortenHome replaces a leading home directory with "~".
func shortenHome(dir, home string) string {
	if home == "" {
		return dir
	}
	if dir == home {
		return "~"
	}
	if strings.HasPrefix(dir, home+"/") {
		return "~" + dir[len(home):]
	}
	return dir
}
