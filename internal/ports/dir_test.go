package ports

import "testing"

func TestParseDirs(t *testing.T) {
	got := parseDirs([]byte("p636\nfcwd\nn/\np21425\nfcwd\nn/Users/me/app\n"))
	if got[636] != "/" || got[21425] != "/Users/me/app" {
		t.Errorf("got %v", got)
	}
}

func TestShortenHome(t *testing.T) {
	cases := map[string]string{
		"/Users/me":         "~",
		"/Users/me/app":     "~/app",
		"/Users/meghan/app": "/Users/meghan/app",
		"/":                 "/",
	}
	for dir, want := range cases {
		if got := shortenHome(dir, "/Users/me"); got != want {
			t.Errorf("shortenHome(%q) = %q, want %q", dir, got, want)
		}
	}
}
