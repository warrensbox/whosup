package ports

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func procInfo(pid int) proc {
	p := proc{uid: -1, cpu: -1, rssKB: -1}
	dir := "/proc/" + strconv.Itoa(pid)

	if b, err := os.ReadFile(dir + "/comm"); err == nil {
		p.command = strings.TrimSpace(string(b))
	}
	if exe, err := os.Readlink(dir + "/exe"); err == nil {
		p.exe = strings.TrimSuffix(exe, " (deleted)")
	} else if b, err := os.ReadFile(dir + "/cmdline"); err == nil {
		if args := parseCmdline(b); len(args) > 0 {
			p.exe = args[0]
		}
	}
	if cwd, err := os.Readlink(dir + "/cwd"); err == nil {
		p.dir = cwd
	}
	if p.command == "" && p.exe != "" {
		p.command = filepath.Base(p.exe)
	}
	if b, err := os.ReadFile(dir + "/status"); err == nil {
		p.uid, p.rssKB = parseStatus(b)
	}
	stat, err1 := os.ReadFile(dir + "/stat")
	uptime, err2 := os.ReadFile("/proc/uptime")
	if err1 == nil && err2 == nil {
		p.cpu = parseCPU(stat, uptime)
	}
	return p
}
