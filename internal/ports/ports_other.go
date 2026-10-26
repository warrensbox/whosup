//go:build !linux && !darwin

package ports

import (
	"errors"
	"runtime"
)

func sockets(bool) ([]socket, error) {
	return nil, errors.New("listing ports is not supported on " + runtime.GOOS)
}

func owners() map[uint64][]int { return nil }

func procInfo(int) proc { return proc{uid: -1, cpu: -1, rssKB: -1} }
