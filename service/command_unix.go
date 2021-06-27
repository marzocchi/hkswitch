// +build linux darwin

package service

import (
	"os/exec"
	"syscall"
)

func init() {
	factory = unixCommandFactory
}

func unixCommandFactory(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd
}