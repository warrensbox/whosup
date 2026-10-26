package ports

import (
	"bytes"
	"encoding/binary"
)

// parseProcargs2 reads the macOS kern.procargs2 sysctl: argc, the
// executable's path, NUL padding, then the arguments and environment.
// It returns the path and the arguments.
func parseProcargs2(b []byte) (string, []string) {
	if len(b) < 4 {
		return "", nil
	}
	argc := int(binary.LittleEndian.Uint32(b))
	b = b[4:]
	exe, b, _ := bytes.Cut(b, []byte{0})
	b = bytes.TrimLeft(b, "\x00")
	var args []string
	for len(args) < argc && len(b) > 0 {
		var arg []byte
		arg, b, _ = bytes.Cut(b, []byte{0})
		args = append(args, string(arg))
	}
	return string(exe), args
}

// parseCmdline splits a Linux /proc/<pid>/cmdline into its arguments.
func parseCmdline(b []byte) []string {
	b = bytes.TrimRight(b, "\x00")
	if len(b) == 0 {
		return nil
	}
	parts := bytes.Split(b, []byte{0})
	args := make([]string, len(parts))
	for i, p := range parts {
		args[i] = string(p)
	}
	return args
}
