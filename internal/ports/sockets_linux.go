package ports

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func sockets(udp bool) ([]socket, error) {
	tables := []string{"tcp", "tcp6"}
	if udp {
		tables = append(tables, "udp", "udp6")
	}
	var result []socket
	for _, name := range tables {
		data, err := os.ReadFile("/proc/net/" + name)
		if errors.Is(err, os.ErrNotExist) {
			continue // IPv6 disabled
		}
		if err != nil {
			return nil, err
		}
		result = append(result, parseProcNet(data, strings.TrimSuffix(name, "6"))...)
	}
	return result, nil
}

// owners maps each socket inode to the PIDs holding it open. Without root
// we can only read our own processes' file descriptors.
func owners() map[uint64][]int {
	result := map[uint64][]int{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return result
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		fdDir := filepath.Join("/proc", e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil || !strings.HasPrefix(link, "socket:[") {
				continue
			}
			inode, err := strconv.ParseUint(strings.TrimSuffix(link[len("socket:["):], "]"), 10, 64)
			if err == nil {
				result[inode] = append(result[inode], pid)
			}
		}
	}
	return result
}
