package ports

import (
	"strconv"
	"strings"
)

// clockTicks is USER_HZ, the unit of times in Linux's /proc/<pid>/stat. It
// is 100 on every Linux architecture Go supports.
const clockTicks = 100

// parseStatus reads the real UID and resident memory from /proc/<pid>/status.
// Either is -1 when missing (kernel threads have no VmRSS).
func parseStatus(b []byte) (uid, rssKB int) {
	uid, rssKB = -1, -1
	for line := range strings.SplitSeq(string(b), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		f := strings.Fields(val)
		if len(f) == 0 {
			continue
		}
		switch key {
		case "Uid":
			if n, err := strconv.Atoi(f[0]); err == nil {
				uid = n
			}
		case "VmRSS":
			if n, err := strconv.Atoi(f[0]); err == nil {
				rssKB = n
			}
		}
	}
	return uid, rssKB
}

// parseCPU works out lifetime CPU usage from /proc/<pid>/stat and
// /proc/uptime, or -1 if they can't be read.
func parseCPU(stat, uptime []byte) float64 {
	// The command (field 2) is in parentheses and may contain spaces.
	s := string(stat)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return -1
	}
	// Fields after the command, starting with state (field 3).
	f := strings.Fields(s[i+1:])
	if len(f) < 20 {
		return -1
	}
	utime, err1 := strconv.ParseFloat(f[11], 64)     // field 14
	stime, err2 := strconv.ParseFloat(f[12], 64)     // field 15
	starttime, err3 := strconv.ParseFloat(f[19], 64) // field 22
	up := strings.Fields(string(uptime))
	if err1 != nil || err2 != nil || err3 != nil || len(up) == 0 {
		return -1
	}
	upSeconds, err := strconv.ParseFloat(up[0], 64)
	if err != nil {
		return -1
	}
	return cpuPercent((utime+stime)/clockTicks, upSeconds-starttime/clockTicks)
}
