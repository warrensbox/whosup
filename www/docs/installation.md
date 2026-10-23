# Installation

## Homebrew

```sh
brew install warrensbox/tap/whosup
```

![Installing whosup](assets/install-whosup.gif)

This also installs shell completions for Bash, Zsh and fish.

To upgrade or remove it:

```sh
brew upgrade whosup
brew uninstall whosup
```

!!! note
Homebrew on Linux needs at least Homebrew 5.0.6 to install casks.

## Install script (Linux)

Installs the latest release into `/usr/local/bin`. It works on macOS too.

```sh
curl -fsSL https://raw.githubusercontent.com/warrensbox/whosup/main/install.sh | sudo sh
```

To install somewhere you can write without `sudo`, pass `-b`:

```sh
curl -fsSL https://raw.githubusercontent.com/warrensbox/whosup/main/install.sh | sh -s -- -b "$HOME/.local/bin"
```

To install a specific version instead of the latest, add the tag. Both options can be combined:

```sh
curl -fsSL https://raw.githubusercontent.com/warrensbox/whosup/main/install.sh | sh -s -- -b "$HOME/.local/bin" v0.0.1
```

The script checks the download against the release's SHA-256 checksums before installing.

## Go

```sh
go install github.com/warrensbox/whosup@latest
```

## From source

```sh
git clone https://github.com/warrensbox/whosup.git
cd whosup
make build
./whosup
```
