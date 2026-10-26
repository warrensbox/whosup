package ports

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
	"strings"
)

// Socket states in /proc/net/{tcp,udp}{,6}.
const (
	tcpListen = "0A"
	udpClose  = "07" // a bound, unconnected UDP socket
)

// parseProcNet reads a /proc/net/{tcp,tcp6,udp,udp6} table and keeps
// listening TCP sockets, or bound and unconnected UDP ones.
func parseProcNet(data []byte, proto string) []socket {
	var result []socket
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Scan() // skip header
	for sc.Scan() {
		// sl local_address rem_address st tx:rx tr:when retrnsmt uid timeout inode
		f := strings.Fields(sc.Text())
		if len(f) < 10 {
			continue
		}
		switch {
		case proto == "tcp" && f[3] == tcpListen:
		case proto == "udp" && f[3] == udpClose && strings.HasSuffix(f[2], ":0000"):
		default:
			continue
		}
		host, port, ok := strings.Cut(f[1], ":")
		if !ok {
			continue
		}
		n, err1 := strconv.ParseUint(port, 16, 16)
		inode, err2 := strconv.ParseUint(f[9], 10, 64)
		addr, ok := procNetAddr(host)
		if err1 != nil || err2 != nil || !ok || inode == 0 || n == 0 {
			continue
		}
		result = append(result, socket{proto: proto, addr: addr, port: int(n), id: inode})
	}
	return result
}

// procNetAddr decodes an address from /proc/net: the IP's 32-bit words in
// hex, each in host (little-endian) byte order.
func procNetAddr(s string) (string, bool) {
	b, err := hex.DecodeString(s)
	if err != nil || (len(b) != net.IPv4len && len(b) != net.IPv6len) {
		return "", false
	}
	for i := 0; i < len(b); i += 4 {
		binary.BigEndian.PutUint32(b[i:], binary.LittleEndian.Uint32(b[i:]))
	}
	return formatAddr(net.IP(b)), true
}

// formatAddr renders a local address the way lsof did: "*" for any
// address, IPv6 in brackets.
func formatAddr(ip net.IP) string {
	switch {
	case ip.IsUnspecified():
		return "*"
	case ip.To4() != nil:
		return ip.To4().String()
	default:
		return "[" + ip.String() + "]"
	}
}
