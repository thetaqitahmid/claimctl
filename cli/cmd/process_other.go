//go:build !unix

package cmd

import (
	"os"
	"os/exec"
)

func prepareProcess(cmd *exec.Cmd) func() { return func() {} }
func signalProcess(cmd *exec.Cmd, sig os.Signal) {
	if cmd.Process != nil {
		_ = cmd.Process.Signal(sig)
	}
}
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
func signalExitCode(sig os.Signal) int        { return ExitCancelled }
func processExitCode(err *exec.ExitError) int { return err.ExitCode() }
