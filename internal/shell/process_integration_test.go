//go:build unix && integration

package shell

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOSRunnerCancellationStopsDescendants(t *testing.T) {
	for _, timeout := range []time.Duration{0, 500 * time.Millisecond} {
		name := "cancel"
		if timeout > 0 {
			name = "timeout"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan int, 1)
			go func() {
				finished <- (OSRunner{}).Run(ctx,
					"sh -c 'echo $$ > child.pid; while :; do printf x >> heartbeat; sleep 0.02; done'; true",
					Options{CWD: dir, Timeout: timeout})
			}()
			// Cleanup also stops a leaked child if the regression returns.
			t.Cleanup(func() {
				if data, err := os.ReadFile(filepath.Join(dir, "child.pid")); err == nil {
					if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
				}
			})
			deadline := time.Now().Add(5 * time.Second)
			for {
				if data, err := os.ReadFile(filepath.Join(dir, "heartbeat")); err == nil && len(data) > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("child did not start")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if timeout == 0 {
				cancel()
			}
			select {
			case code := <-finished:
				if code == 0 {
					t.Fatal("canceled command succeeded")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("canceled command did not return")
			}
			before, err := os.ReadFile(filepath.Join(dir, "heartbeat"))
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(100 * time.Millisecond)
			after, err := os.ReadFile(filepath.Join(dir, "heartbeat"))
			if err != nil || string(before) != string(after) {
				t.Fatalf("child continued after cancellation: before = %q, after = %q, error = %v", before, after, err)
			}
		})
	}
}
