package ports

import (
	"encoding/binary"
	"net"
)

// Offsets into the records of the macOS sysctls net.inet.{tcp,udp}.pcblist64,
// from xnu's bsd/netinet/in_pcb.h (struct xinpcb64, packed to 4 bytes) and
// bsd/sys/socketvar.h (struct xsocket64). A TCP record is a struct xtcpcb64,
// which starts with a 4-byte length and then the xinpcb64; a UDP record is
// the xinpcb64 itself.
const (
	xinpgenLen = 24 // the struct xinpgen header and trailer

	tcpInpcb = 4 // where the xinpcb64 starts in a TCP record
	udpInpcb = 0

	inpFport  = 16  // foreign port, network byte order
	inpLport  = 18  // local port, network byte order
	inpVflag  = 92  // INP_IPV4 / INP_IPV6
	inpLaddr  = 112 // in6_addr, or in_addr_4in6 with the IPv4 address at +12
	inpSocket = 148 // struct xsocket64

	xsoSo      = 0  // kernel socket handle, as libproc's soi_so
	xsoOptions = 10 // so_options

	inpIPv6      = 0x2
	soAcceptConn = 0x2 // SO_ACCEPTCONN: a listening socket
)

// parsePcblist reads a pcblist64 sysctl and keeps listening TCP sockets,
// or unconnected UDP ones.
func parsePcblist(b []byte, proto string) []socket {
	base := udpInpcb
	if proto == "tcp" {
		base = tcpInpcb
	}
	le := binary.LittleEndian
	var result []socket
	if len(b) < xinpgenLen {
		return nil
	}
	for off := int(le.Uint32(b)); off+4 <= len(b); {
		n := int(le.Uint32(b[off:]))
		if n <= xinpgenLen || off+n > len(b) {
			break // the trailing xinpgen, or garbage
		}
		r := b[off : off+n]
		off += n
		inp := base
		so := inp + inpSocket
		if len(r) < so+xsoOptions+2 {
			continue
		}
		if proto == "tcp" && le.Uint16(r[so+xsoOptions:])&soAcceptConn == 0 {
			continue
		}
		if proto == "udp" && binary.BigEndian.Uint16(r[inp+inpFport:]) != 0 {
			continue
		}
		port := int(binary.BigEndian.Uint16(r[inp+inpLport:]))
		if port == 0 {
			continue // not bound yet
		}
		var ip net.IP
		if r[inp+inpVflag]&inpIPv6 != 0 {
			ip = net.IP(append([]byte(nil), r[inp+inpLaddr:inp+inpLaddr+16]...))
		} else {
			ip = net.IP(append([]byte(nil), r[inp+inpLaddr+12:inp+inpLaddr+16]...))
		}
		result = append(result, socket{
			proto: proto,
			addr:  formatAddr(ip),
			port:  port,
			id:    le.Uint64(r[so+xsoSo:]),
		})
	}
	return result
}
