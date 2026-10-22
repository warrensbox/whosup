package ports

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Stop asks the process to shut down gracefully by sending SIGTERM.
func Stop(pid int) error { return signal(pid, syscall.SIGTERM) }

// Kill forces the process to exit immediately by sending SIGKILL.
func Kill(pid int) error { return signal(pid, syscall.SIGKILL) }

// Alive reports whether a process with this PID still exists.
func Alive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	// EPERM means it exists but belongs to someone else.
	return err == nil || errors.Is(err, syscall.EPERM)
}

func signal(pid int, sig syscall.Signal) error {
	if pid <= 1 || pid == os.Getpid() {
		return fmt.Errorf("refusing to stop PID %d", pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}
