# Installation

## Homebrew

```sh
brew install --cask warrensbox/tap/whosup
```

![Installing whosup](assets/images/install-whosup.gif)

This also installs shell completions for Bash, Zsh and fish.

To upgrade or remove it:

```sh
brew upgrade --cask whosup
brew uninstall --cask whosup
```

!!! note
Homebrew on Linux needs at least Homebrew 5.0.6 to install casks.

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
