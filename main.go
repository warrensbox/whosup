package main

import "github.com/warrensbox/whosup/cmd"

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(version)
}
