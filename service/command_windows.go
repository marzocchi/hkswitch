// +build windows

package service

import (
	"os/exec"
	"syscall"
)

func init() {
	factory = windowsCommandFactory
}

func windowsCommandFactory(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	return cmd
}
