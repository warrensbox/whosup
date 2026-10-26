package ports

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
)

// Port is a listening TCP port (or bound UDP port) and the process that owns it.
type Port struct {
	Port    int
	Proto   string // "tcp" or "udp"
	PID     int
	Command string
	Address string
	CPU     string
	Memory  string
	Dir     string
	User    string // owner of the process
	Exe     string // path to the executable, if known
}

// Label is the port number, with "/udp" appended for UDP ports.
func (p Port) Label() string {
	if p.Proto == "udp" {
		return strconv.Itoa(p.Port) + "/udp"
	}
	return strconv.Itoa(p.Port)
}

// socket is a listening socket read from the kernel's tables.
type socket struct {
	proto string
	addr  string
	port  int
	id    uint64 // inode on Linux, kernel socket handle on macOS
}

// proc is what we know about a process. Unknown fields are empty, and
// cpu and rssKB are negative.
type proc struct {
	command string
	uid     int
	exe     string
	dir     string
	cpu     float64 // percent of one CPU, averaged over the process's lifetime
	rssKB   int
}

// List returns listening TCP ports, and bound UDP ports too if udp is set,
// read straight from the kernel. Sockets whose owner we can't see (another
// user's, without sudo) are left out.
func List(udp bool) ([]Port, error) {
	socks, err := sockets(udp)
	if err != nil {
		return nil, fmt.Errorf("reading sockets: %w", err)
	}
	if len(socks) == 0 {
		return nil, nil
	}
	pids := owners()
	home, err := os.UserHomeDir()
	if err != nil {
		home = "" // leave paths unshortened
	}
	procs := map[int]proc{}
	seen := map[string]bool{}
	var result []Port
	for _, s := range socks {
		// A socket is shared by every process that inherited it.
		for _, pid := range pids[s.id] {
			key := fmt.Sprint(s.proto, pid, s.addr, s.port)
			if seen[key] {
				continue
			}
			seen[key] = true
			pr, ok := procs[pid]
			if !ok {
				pr = procInfo(pid)
				procs[pid] = pr
			}
			result = append(result, toPort(s, pid, pr, home))
		}
	}
	sort.SliceStable(result, func(a, b int) bool {
		if result[a].Port != result[b].Port {
			return result[a].Port < result[b].Port
		}
		return result[a].Proto < result[b].Proto
	})
	return result, nil
}

func toPort(s socket, pid int, pr proc, home string) Port {
	p := Port{Port: s.port, Proto: s.proto, PID: pid, Address: s.addr, Command: pr.command, Exe: pr.exe,
		User: userName(pr.uid), CPU: "-", Memory: "-", Dir: "-"}
	if p.Command == "" {
		p.Command = "-"
	}
	if pr.cpu >= 0 {
		p.CPU = fmt.Sprintf("%.1f%%", pr.cpu)
	}
	if pr.rssKB >= 0 {
		p.Memory = formatKB(pr.rssKB)
	}
	if pr.dir != "" {
		p.Dir = shortenHome(pr.dir, home)
	}
	return p
}

var userNames = map[int]string{}

// userName looks up a UID's name, caching the answer. Unknown UIDs are
// shown as numbers, and a negative UID as "".
func userName(uid int) string {
	if uid < 0 {
		return ""
	}
	if name, ok := userNames[uid]; ok {
		return name
	}
	name := strconv.Itoa(uid)
	if u, err := user.LookupId(name); err == nil {
		name = u.Username
	}
	userNames[uid] = name
	return name
}

// cpuPercent is CPU time as a percentage of elapsed time, like ps.
func cpuPercent(cpuSeconds, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return 100 * cpuSeconds / elapsedSeconds
}
