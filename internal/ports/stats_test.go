package ports

import "testing"

func TestParseStats(t *testing.T) {
	got := parseStats([]byte("  636   0.3  51200 /usr/libexec/rapportd\n21425  12.5 204800 /Applications/Code Helper (Plugin)\nbad line here\n"))
	if s := got[636]; s.cpu != 0.3 || s.rssKB != 51200 || s.exe != "/usr/libexec/rapportd" {
		t.Errorf("pid 636: got %+v", s)
	}
	if s := got[21425]; s.cpu != 12.5 || s.rssKB != 204800 || s.exe != "/Applications/Code Helper (Plugin)" {
		t.Errorf("pid 21425: got %+v", s)
	}
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
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
