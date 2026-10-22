package ports

import "testing"

func TestParse(t *testing.T) {
	out := []byte(`COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
node     4321 me     23u  IPv4 0x1      0t0  TCP 127.0.0.1:3000 (LISTEN)
postgres  111 me      7u  IPv6 0x2      0t0  TCP [::1]:5432 (LISTEN)
postgres  111 me      8u  IPv6 0x2      0t0  TCP [::1]:5432 (LISTEN)
`)
	got := parse(out)
	if len(got) != 2 {
		t.Fatalf("got %d ports, want 2: %+v", len(got), got)
	}
	if got[0].Port != 3000 || got[0].Command != "node" || got[0].PID != 4321 || got[0].User != "me" {
		t.Errorf("unexpected first port: %+v", got[0])
	}
	if got[1].Port != 5432 || got[1].Address != "[::1]" {
		t.Errorf("unexpected second port: %+v", got[1])
	}
}
