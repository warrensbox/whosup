package ports

import (
	"bufio"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestRefusesSpecialPIDs(t *testing.T) {
	for _, pid := range []int{0, 1, os.Getpid()} {
		if Stop(pid) == nil || Kill(pid) == nil {
			t.Errorf("expected refusal for PID %d", pid)
		}
	}
}

func TestStopAndKill(t *testing.T) {
	// A child that ignores SIGTERM, so only Kill ends it.
	cmd := exec.Command("sh", "-c", "trap '' TERM; echo ready; while :; do sleep 0.1; done")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Wait until the trap is installed before signalling.
	if _, err := bufio.NewReader(out).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }() //nolint:errcheck // the child is killed, so Wait always errors

	if !Alive(pid) {
		t.Fatal("child should be alive")
	}
	if err := Stop(pid); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
		t.Fatal("child should have ignored SIGTERM")
	case <-time.After(300 * time.Millisecond):
	}
	if err := Kill(pid); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("child did not exit after SIGKILL")
	}
	if Alive(pid) {
		t.Error("child should be gone")
	}
}
