package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/warrensbox/whosup/internal/ports"
	"github.com/warrensbox/whosup/internal/ui"
)

var (
	port    int  // set by -p/--port to show details for a single port
	pidOnly bool // set by -q/--pid to print only the PIDs on that port
	mine    bool // set by -u/--user to hide other users' and OS processes
)

var rootCmd = &cobra.Command{
	Use:   "whosup",
	Short: "whosup shows which ports are in use",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if pidOnly && !cmd.Flags().Changed("port") {
			return fmt.Errorf("--pid requires --port")
		}
		if cmd.Flags().Changed("port") && (port < 1 || port > 65535) {
			return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
		}
		if cmd.Flags().Changed("port") {
			// Asking for a port always answers, whoever owns it.
			list, err := ports.List()
			if err != nil {
				return err
			}
			return showPort(cmd, list, port)
		}
		load := ports.List
		if mine {
			load = func() ([]ports.Port, error) {
				list, err := ports.List()
				return ports.UserStarted(list), err
			}
		}
		list, err := load()
		if err != nil {
			return err
		}
		if len(list) == 0 {
			if mine {
				cmd.Println("No listening ports found for services you started. Drop --user to see all.")
			} else {
				cmd.Println("No listening ports found.")
			}
			return nil
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			return ui.Run(list, load)
		}
		// Not a terminal (e.g. piped): print a static table.
		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStdout())
		t.SetStyle(table.StyleRounded)
		t.AppendHeader(table.Row{"Port", "PID", "Command", "Address", "CPU", "Memory", "Directory"})
		for _, p := range list {
			t.AppendRow(table.Row{p.Port, p.PID, p.Command, p.Address, p.CPU, p.Memory, p.Dir})
		}
		t.Render()
		return nil
	},
}

// showPort prints the details of every process listening on port, or just
// their PIDs with --pid. Output goes to stdout so it can be piped.
func showPort(cmd *cobra.Command, list []ports.Port, port int) error {
	out := cmd.OutOrStdout()
	seen := map[int]bool{}
	for _, p := range list {
		if p.Port != port {
			continue
		}
		if pidOnly {
			// A process listening on both IPv4 and IPv6 appears twice.
			if !seen[p.PID] {
				fmt.Fprintln(out, p.PID)
			}
		} else {
			if len(seen) > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprint(out, ui.Details(p))
		}
		seen[p.PID] = true
	}
	if len(seen) == 0 {
		cmd.PrintErrf("Nothing is listening on port %d.\n", port)
		if pidOnly {
			// Like lsof -t: no output and a non-zero exit when nothing matches.
			os.Exit(1)
		}
	}
	return nil
}

func init() {
	// Keep "whosup completion <shell>" working, but out of --help.
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "show details for a single port")
	rootCmd.Flags().BoolVarP(&pidOnly, "pid", "q", false, "with --port, prints the PID(s), one per line")
	rootCmd.Flags().BoolVarP(&mine, "user", "u", false, "show only services you started (hide OS services)")
}

// Execute runs the root command. version is shown by --version.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
