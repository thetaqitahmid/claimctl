//go:build unix

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecSignalHelper(t *testing.T) {
	if os.Getenv("CLAIMCTL_SIGNAL_HELPER") != "1" {
		return
	}
	rootCmd.SetArgs([]string{"exec", "gpu", "--netrc=false", "--lease-duration=2s", "--heartbeat-interval=100ms", "--kill-grace=500ms", "--", "sh", "-c", `trap 'printf INT > "$CLAIMCTL_SIGNAL_FILE"; exit 0' INT; trap 'printf TERM > "$CLAIMCTL_SIGNAL_FILE"; exit 0' TERM; printf ready > "$CLAIMCTL_READY_FILE"; while :; do sleep 0.05; done`})
	Execute()
	os.Exit(0)
}

func TestExecForwardsSignalsAndReleases(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			address, _, completed, _ := execFixture(t, "active", 200)
			directory := t.TempDir()
			ready := filepath.Join(directory, "ready")
			received := filepath.Join(directory, "signal")
			helper := exec.Command(os.Args[0], "-test.run=^TestExecSignalHelper$")
			helper.Env = append(os.Environ(), "CLAIMCTL_SIGNAL_HELPER=1", "CLAIMCTL_URL="+address, "CLAIMCTL_TOKEN=secret", "CLAIMCTL_TOKEN_FILE=", "CLAIMCTL_CA_CERT=", "CLAIMCTL_JSON=false", "XDG_CONFIG_HOME="+directory, "CLAIMCTL_READY_FILE="+ready, "CLAIMCTL_SIGNAL_FILE="+received)
			require.NoError(t, helper.Start())
			t.Cleanup(func() { _ = helper.Process.Kill() })
			done := make(chan error, 1)
			go func() { done <- helper.Wait() }()
			require.Eventually(t, func() bool { _, err := os.Stat(ready); return err == nil }, 3*time.Second, 10*time.Millisecond)
			require.NoError(t, helper.Process.Signal(sig))
			select {
			case err := <-done:
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				require.Equal(t, 128+int(sig), exit.ExitCode())
			case <-time.After(3 * time.Second):
				t.Fatal("exec did not stop after signal")
			}
			data, err := os.ReadFile(received)
			require.NoError(t, err)
			expected := "INT"
			if sig == syscall.SIGTERM {
				expected = "TERM"
			}
			require.Equal(t, expected, string(data))
			require.Equal(t, int32(1), completed.Load())
		})
	}
}
