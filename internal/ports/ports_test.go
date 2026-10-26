package ports

import (
	"encoding/binary"
	"net"
	"os"
	"reflect"
	"testing"
)

func TestParseProcNet(t *testing.T) {
	tcp := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 4321 1 0000000000000000 100 0 0 10 0
   1: 00000000:1538 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 111 1 0000000000000000 100 0 0 10 0
   2: 0100007F:0BB8 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 999 1 0000000000000000 20 4 30 10 -1
`)
	got := parseProcNet(tcp, "tcp")
	want := []socket{
		{proto: "tcp", addr: "127.0.0.1", port: 3000, id: 4321},
		{proto: "tcp", addr: "*", port: 5432, id: 111},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tcp: got %+v, want %+v", got, want)
	}

	tcp6 := []byte(`  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000001000000:1538 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 222 1 0000000000000000 100 0 0 10 0
   1: 00000000000000000000000000000000:1F90 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 333 1 0000000000000000 100 0 0 10 0
`)
	got = parseProcNet(tcp6, "tcp")
	want = []socket{
		{proto: "tcp", addr: "[::1]", port: 5432, id: 222},
		{proto: "tcp", addr: "*", port: 8080, id: 333},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tcp6: got %+v, want %+v", got, want)
	}

	udp := []byte(`   sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
  100: 3500007F:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000   101        0 555 2 0000000000000000 0
  101: 0100007F:C350 0100007F:0035 01 00000000:00000000 00:00000000 00000000  1000        0 666 2 0000000000000000 0
`)
	got = parseProcNet(udp, "udp")
	want = []socket{{proto: "udp", addr: "127.0.0.53", port: 53, id: 555}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("udp: got %+v, want %+v", got, want)
	}
}

// pcbRecord builds one pcblist64 record the way xnu lays it out.
func pcbRecord(proto string, lport, fport uint16, ip net.IP, so uint64, options uint16) []byte {
	base := udpInpcb
	if proto == "tcp" {
		base = tcpInpcb
	}
	r := make([]byte, base+inpSocket+64)
	binary.LittleEndian.PutUint32(r, uint32(len(r)))
	binary.BigEndian.PutUint16(r[base+inpLport:], lport)
	binary.BigEndian.PutUint16(r[base+inpFport:], fport)
	if ip4 := ip.To4(); ip4 != nil {
		r[base+inpVflag] = 0x1
		copy(r[base+inpLaddr+12:], ip4)
	} else {
		r[base+inpVflag] = inpIPv6
		copy(r[base+inpLaddr:], ip)
	}
	binary.LittleEndian.PutUint64(r[base+inpSocket+xsoSo:], so)
	binary.LittleEndian.PutUint16(r[base+inpSocket+xsoOptions:], options)
	return r
}

func pcblist(records ...[]byte) []byte {
	gen := make([]byte, xinpgenLen)
	binary.LittleEndian.PutUint32(gen, xinpgenLen)
	b := append([]byte(nil), gen...)
	for _, r := range records {
		b = append(b, r...)
	}
	return append(b, gen...)
}

func TestParsePcblist(t *testing.T) {
	tcp := pcblist(
		pcbRecord("tcp", 3000, 0, net.ParseIP("127.0.0.1"), 0xa, soAcceptConn|0x4),
		pcbRecord("tcp", 51000, 3000, net.ParseIP("127.0.0.1"), 0xb, 0), // a connection
		pcbRecord("tcp", 5432, 0, net.ParseIP("::1"), 0xc, soAcceptConn),
		pcbRecord("tcp", 8080, 0, net.IPv6unspecified, 0xd, soAcceptConn),
	)
	got := parsePcblist(tcp, "tcp")
	want := []socket{
		{proto: "tcp", addr: "127.0.0.1", port: 3000, id: 0xa},
		{proto: "tcp", addr: "[::1]", port: 5432, id: 0xc},
		{proto: "tcp", addr: "*", port: 8080, id: 0xd},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tcp: got %+v, want %+v", got, want)
	}

	udp := pcblist(
		pcbRecord("udp", 5353, 0, net.IPv4zero, 0x1, 0),
		pcbRecord("udp", 60000, 53, net.ParseIP("10.0.0.2"), 0x2, 0), // connected
		pcbRecord("udp", 0, 0, net.IPv4zero, 0x3, 0),                 // not bound yet
	)
	got = parsePcblist(udp, "udp")
	want = []socket{{proto: "udp", addr: "*", port: 5353, id: 0x1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("udp: got %+v, want %+v", got, want)
	}

	if got := parsePcblist([]byte{1, 2}, "tcp"); got != nil {
		t.Errorf("short input: got %+v", got)
	}
}

func TestParseProcargs2(t *testing.T) {
	b := []byte("\x02\x00\x00\x00/usr/local/bin/node\x00\x00\x00\x00node\x00server.js\x00HOME=/Users/me\x00")
	exe, args := parseProcargs2(b)
	if exe != "/usr/local/bin/node" || !reflect.DeepEqual(args, []string{"node", "server.js"}) {
		t.Errorf("got %q %q", exe, args)
	}
}

func TestParseCmdline(t *testing.T) {
	if got := parseCmdline([]byte("python3\x00-m\x00http.server\x00")); !reflect.DeepEqual(got, []string{"python3", "-m", "http.server"}) {
		t.Errorf("got %q", got)
	}
	if got := parseCmdline(nil); got != nil {
		t.Errorf("empty: got %q", got)
	}
}

func TestParseStatus(t *testing.T) {
	uid, rss := parseStatus([]byte("Name:\tnode\nUid:\t1000\t1000\t1000\t1000\nVmRSS:\t   20480 kB\n"))
	if uid != 1000 || rss != 20480 {
		t.Errorf("got uid %d rss %d", uid, rss)
	}
	if uid, rss := parseStatus([]byte("Name:\tkthreadd\nUid:\t0\t0\t0\t0\n")); uid != 0 || rss != -1 {
		t.Errorf("kernel thread: got uid %d rss %d", uid, rss)
	}
}

func TestParseCPU(t *testing.T) {
	// 150 + 50 ticks of CPU, started at tick 1000, uptime 30 s: 2 s of 20 s.
	stat := []byte("4321 (my (odd) app) S 1 4321 4321 0 -1 4194304 1 0 0 0 150 50 0 0 20 0 1 0 1000 1000 200 18446744073709551615")
	if got := parseCPU(stat, []byte("30.00 100.00\n")); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	if got := parseCPU([]byte("garbage"), []byte("30.00 100.00\n")); got != -1 {
		t.Errorf("garbage: got %v, want -1", got)
	}
}

func TestLabel(t *testing.T) {
	if got := (Port{Port: 3000, Proto: "tcp"}).Label(); got != "3000" {
		t.Errorf("tcp: got %q", got)
	}
	if got := (Port{Port: 53, Proto: "udp"}).Label(); got != "53/udp" {
		t.Errorf("udp: got %q", got)
	}
}

func TestFormatKB(t *testing.T) {
	cases := map[int]string{
		512:         "512 KB",
		2048:        "2.0 MB",
		1536 * 1024: "1.5 GB",
	}
	for kb, want := range cases {
		if got := formatKB(kb); got != want {
			t.Errorf("formatKB(%d) = %q, want %q", kb, got, want)
		}
	}
}

// TestListFindsOwnSockets reads the real kernel tables.
func TestListFindsOwnSockets(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	u, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	tcpPort := l.Addr().(*net.TCPAddr).Port
	udpPort := u.LocalAddr().(*net.UDPAddr).Port

	find := func(list []Port, proto string, port int) *Port {
		for i := range list {
			if list[i].Proto == proto && list[i].Port == port && list[i].PID == os.Getpid() {
				return &list[i]
			}
		}
		return nil
	}

	list, err := List(true)
	if err != nil {
		t.Fatal(err)
	}
	p := find(list, "tcp", tcpPort)
	if p == nil {
		t.Fatalf("TCP port %d not found in %+v", tcpPort, list)
	}
	if p.Address != "127.0.0.1" || p.Command == "-" || p.Exe == "" || p.User == "" || p.Dir == "-" || p.CPU == "-" || p.Memory == "-" {
		t.Errorf("incomplete TCP entry: %+v", *p)
	}
	if find(list, "udp", udpPort) == nil {
		t.Errorf("UDP port %d not found with udp", udpPort)
	}

	list, err = List(false)
	if err != nil {
		t.Fatal(err)
	}
	if find(list, "tcp", tcpPort) == nil {
		t.Errorf("TCP port %d not found without udp", tcpPort)
	}
	for _, p := range list {
		if p.Proto == "udp" {
			t.Fatalf("UDP entry without udp: %+v", p)
		}
	}
}
