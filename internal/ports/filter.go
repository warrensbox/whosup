package ports

import (
	"os"
	"os/user"
	"strings"
)

// systemDirs hold programs that ship with the operating system.
var systemDirs = []string{"/System/", "/usr/libexec/", "/usr/sbin/", "/sbin/", "/usr/lib/", "/lib/"}

// UserStarted keeps the ports owned by the current user (the one who ran
// sudo, under sudo) whose executables aren't part of the operating system.
func UserStarted(list []Port) []Port {
	me := os.Getenv("SUDO_USER")
	if me == "" {
		if u, err := user.Current(); err == nil {
			me = u.Username
		}
	}
	var result []Port
	for _, p := range list {
		if !isSystem(p, me) {
			result = append(result, p)
		}
	}
	return result
}

// isSystem reports whether p belongs to another user or the operating system.
func isSystem(p Port, me string) bool {
	if me != "" && p.User != "" && p.User != me {
		return true
	}
	for _, dir := range systemDirs {
		if strings.HasPrefix(p.Exe, dir) {
			return true
		}
	}
	return false
}
