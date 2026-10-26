package ports

import "testing"

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
