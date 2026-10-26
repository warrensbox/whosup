package ports

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

const (
	taskInfoLen   = 96   // struct proc_taskinfo
	vnodePathLen  = 2352 // struct proc_vnodepathinfo
	cdirPathOff   = 152  // pvi_cdir.vip_path, after struct vnode_info
	pathInfoLen   = 4096 // PROC_PIDPATHINFO_MAXSIZE
	maxPathLen    = 1024 // MAXPATHLEN
	taskRSSOff    = 8    // pti_resident_size, bytes
	taskUserOff   = 16   // pti_total_user, Mach time units
	taskSystemOff = 24   // pti_total_system, Mach time units
)

// machTimeNs converts Mach time units to nanoseconds: Apple silicon's timer
// runs at 24 MHz (mach_timebase_info 125/3), Intel's is in nanoseconds.
func machTimeNs(t uint64) float64 {
	if runtime.GOARCH == "arm64" {
		return float64(t) * 125 / 3
	}
	return float64(t)
}

func procInfo(pid int) proc {
	p := proc{uid: -1, cpu: -1, rssKB: -1}

	buf := make([]byte, pathInfoLen)
	if _, err := procInfoCall(procInfoPidInfo, pid, pidPathInfo, 0, buf); err == nil {
		p.exe = cString(buf)
	}
	if p.exe == "" {
		// procargs2 only works for our own processes, but try it anyway.
		if b, err := unix.SysctlRaw("kern.procargs2", pid); err == nil {
			p.exe, _ = parseProcargs2(b)
		}
	}
	if p.exe != "" {
		p.command = filepath.Base(p.exe)
	}

	var start time.Time
	if k, err := unix.SysctlKinfoProc("kern.proc.pid", pid); err == nil {
		p.uid = int(k.Eproc.Ucred.Uid)
		start = time.Unix(k.Proc.P_starttime.Unix())
		if p.command == "" {
			p.command = cString(k.Proc.P_comm[:])
		}
	}

	buf = make([]byte, vnodePathLen)
	if n, err := procInfoCall(procInfoPidInfo, pid, pidVnodePathInfo, 0, buf); err == nil && n >= cdirPathOff+maxPathLen {
		p.dir = cString(buf[cdirPathOff : cdirPathOff+maxPathLen])
	}

	buf = make([]byte, taskInfoLen)
	if n, err := procInfoCall(procInfoPidInfo, pid, pidTaskInfo, 0, buf); err == nil && n >= taskSystemOff+8 {
		le := binary.LittleEndian
		p.rssKB = int(le.Uint64(buf[taskRSSOff:]) / 1024)
		if !start.IsZero() {
			cpu := machTimeNs(le.Uint64(buf[taskUserOff:]) + le.Uint64(buf[taskSystemOff:]))
			p.cpu = cpuPercent(cpu/1e9, time.Since(start).Seconds())
		}
	}
	return p
}

// cString returns b up to its first NUL.
func cString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
