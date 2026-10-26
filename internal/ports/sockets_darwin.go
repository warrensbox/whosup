package ports

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

// proc_info(2) calls and flavors, from xnu's bsd/sys/proc_info.h. This is
// the system call that libproc's proc_listpids, proc_pidinfo and
// proc_pidfdinfo wrap.
const (
	procInfoListPids  = 1 // PROC_INFO_CALL_LISTPIDS
	procInfoPidInfo   = 2 // PROC_INFO_CALL_PIDINFO
	procInfoPidFdInfo = 3 // PROC_INFO_CALL_PIDFDINFO

	procAllPids = 1 // PROC_ALL_PIDS

	pidListFds        = 1  // PROC_PIDLISTFDS: []struct proc_fdinfo
	pidTaskInfo       = 4  // PROC_PIDTASKINFO: struct proc_taskinfo
	pidVnodePathInfo  = 9  // PROC_PIDVNODEPATHINFO: struct proc_vnodepathinfo
	pidPathInfo       = 11 // PROC_PIDPATHINFO: the executable's path
	pidFdSocketInfo   = 3  // PROC_PIDFDSOCKETINFO: struct socket_fdinfo
	fdTypeSocket      = 2  // PROX_FDTYPE_SOCKET
	fdInfoLen         = 8  // struct proc_fdinfo: int32 fd, uint32 type
	socketFdInfoLen   = 792
	socketFdInfoSoOff = 160 // soi_so, after proc_fileinfo (24) and vinfo_stat (136)
)

// procInfoCall calls proc_info(2) and returns how many bytes it wrote.
func procInfoCall(call, pid, flavor int, arg uint64, buf []byte) (int, error) {
	var p unsafe.Pointer
	if len(buf) > 0 {
		p = unsafe.Pointer(&buf[0])
	}
	r, _, errno := unix.Syscall6(unix.SYS_PROC_INFO, uintptr(call), uintptr(pid), uintptr(flavor),
		uintptr(arg), uintptr(p), uintptr(len(buf)))
	if errno != 0 {
		return 0, errno
	}
	return int(r), nil
}

func sockets(udp bool) ([]socket, error) {
	tables := []string{"tcp"}
	if udp {
		tables = append(tables, "udp")
	}
	var result []socket
	for _, proto := range tables {
		b, err := unix.SysctlRaw("net.inet." + proto + ".pcblist64")
		if err != nil {
			return nil, err
		}
		result = append(result, parsePcblist(b, proto)...)
	}
	return result, nil
}

// owners maps each kernel socket handle to the PIDs holding it open. Without
// root we can only see our own processes' file descriptors.
func owners() map[uint64][]int {
	result := map[uint64][]int{}
	for _, pid := range allPids() {
		for _, so := range socketsOf(pid) {
			result[so] = append(result[so], pid)
		}
	}
	return result
}

func allPids() []int {
	n, err := procInfoCall(procInfoListPids, procAllPids, 0, 0, nil)
	if err != nil || n <= 0 {
		return nil
	}
	buf := make([]byte, n+64*4) // room for processes started since
	n, err = procInfoCall(procInfoListPids, procAllPids, 0, 0, buf)
	if err != nil {
		return nil
	}
	var pids []int
	for i := 0; i+4 <= n; i += 4 {
		if pid := int(int32(binary.LittleEndian.Uint32(buf[i:]))); pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// socketsOf lists the kernel handles of a process's open sockets.
func socketsOf(pid int) []uint64 {
	n, err := procInfoCall(procInfoPidInfo, pid, pidListFds, 0, nil)
	if err != nil || n <= 0 {
		return nil
	}
	buf := make([]byte, n+32*fdInfoLen) // room for files opened since
	if n, err = procInfoCall(procInfoPidInfo, pid, pidListFds, 0, buf); err != nil {
		return nil
	}
	le := binary.LittleEndian
	var result []uint64
	info := make([]byte, socketFdInfoLen)
	for i := 0; i+fdInfoLen <= n; i += fdInfoLen {
		if le.Uint32(buf[i+4:]) != fdTypeSocket {
			continue
		}
		fd := uint64(le.Uint32(buf[i:]))
		m, err := procInfoCall(procInfoPidFdInfo, pid, pidFdSocketInfo, fd, info)
		if err != nil || m < socketFdInfoSoOff+8 {
			continue // closed since, or not ours to read
		}
		result = append(result, le.Uint64(info[socketFdInfoSoOff:]))
	}
	return result
}
