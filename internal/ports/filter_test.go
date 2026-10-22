package ports

import "testing"

func TestIsSystem(t *testing.T) {
	cases := []struct {
		p    Port
		want bool
	}{
		{Port{User: "me", Exe: "/opt/homebrew/bin/python3"}, false},
		{Port{User: "me", Exe: "/Applications/Postgres.app/Contents/Versions/17/bin/postgres"}, false},
		{Port{User: "me", Exe: "node"}, false},
		{Port{User: "me", Exe: ""}, false},
		{Port{User: "me", Exe: "/System/Library/CoreServices/ControlCenter.app/Contents/MacOS/ControlCenter"}, true},
		{Port{User: "me", Exe: "/usr/libexec/rapportd"}, true},
		{Port{User: "me", Exe: "/usr/sbin/sshd"}, true},
		{Port{User: "root", Exe: "/opt/homebrew/bin/nginx"}, true},
	}
	for _, c := range cases {
		if got := isSystem(c.p, "me"); got != c.want {
			t.Errorf("isSystem(%+v) = %v, want %v", c.p, got, c.want)
		}
	}
}
