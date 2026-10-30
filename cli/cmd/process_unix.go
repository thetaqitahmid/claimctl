//go:build unix

package cmd

import (
	"golang.org/x/sys/unix"
	"golang.org/x/term"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func prepareProcess(cmd *exec.Cmd) func() {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	input, ok := cmd.Stdin.(*os.File)
	if !ok || !term.IsTerminal(int(input.Fd())) {
		return func() {}
	}
	fd := int(input.Fd())
	foreground, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil || foreground != syscall.Getpgrp() {
		return func() {}
	}
	cmd.SysProcAttr.Foreground = true
	cmd.SysProcAttr.Ctty = fd
	return func() {
		signal.Ignore(syscall.SIGTTOU)
		_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, foreground)
		signal.Reset(syscall.SIGTTOU)
	}
}
func signalProcess(cmd *exec.Cmd, sig os.Signal) {
	if cmd.Process != nil {
		if value, ok := sig.(syscall.Signal); ok {
			_ = syscall.Kill(-cmd.Process.Pid, value)
		}
	}
}
func killProcess(cmd *exec.Cmd) { signalProcess(cmd, syscall.SIGKILL) }
func signalExitCode(sig os.Signal) int {
	if value, ok := sig.(syscall.Signal); ok {
		return 128 + int(value)
	}
	return ExitCancelled
}
func processExitCode(err *exec.ExitError) int {
	if status, ok := err.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return err.ExitCode()
}
