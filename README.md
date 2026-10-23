# whosup

See what's running on your ports, and stop it.
Because nobody remembers lsof -ti:8080 | xargs kill.

## Install

### Homebrew

```sh
brew install --cask warrensbox/tap/whosup
```

![Installing whosup](www/docs/assets/install-whosup.gif)

This also installs shell completions for Bash, Zsh and fish.

To upgrade or remove it:

```sh
brew upgrade --cask whosup
brew uninstall --cask whosup
```

> **Note:** Homebrew on Linux needs at least Homebrew 5.0.6 to install casks.

### Go

```sh
go install github.com/warrensbox/whosup@latest
```

### From source

```sh
git clone https://github.com/warrensbox/whosup.git
cd whosup
make build
./whosup
```

## Usage

```sh
whosup                # interactive table
whosup -u             # only services you started (also --user)
whosup -p 8080        # show what's on one port (also --port)
whosup -p 8080 -q     # print only the PID(s) on that port (also --pid)
```

![Stopping a service on a local port with whosup](www/docs/assets/stop-port-local-whosup.gif)

### Keys

| Key         | Action                      |
| ----------- | --------------------------- |
| `↑` / `k`   | Move up                     |
| `↓` / `j`   | Move down                   |
| `m`         | Show the row's full details |
| `r`         | Refresh the list            |
| `enter`     | Stop the selected service   |
| `q` / `esc` | Quit                        |

In the stop dialog, use `←` / `→` to choose, `enter` to confirm, or `y` / `n` as shortcuts. The dialog defaults to **No**.

## Notes

- `-u` / `--user` hides processes owned by other users and programs that ship with the OS (anything under `/System`, `/usr/libexec`, `/usr/sbin`, `/sbin`, `/usr/lib` or `/lib`). `-p` ignores it and always shows whatever is on the port.
- Without `sudo`, only your own processes are shown in full. Others may be missing or show `-`.
- Only TCP ports that are listening are shown. UDP isn't included.
