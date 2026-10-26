package ports

import "strings"

// shortenHome replaces a leading home directory with "~".
func shortenHome(dir, home string) string {
	if home == "" {
		return dir
	}
	if dir == home {
		return "~"
	}
	if strings.HasPrefix(dir, home+"/") {
		return "~" + dir[len(home):]
	}
	return dir
}
