package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var stopSignals = map[string]syscall.Signal{
	"HUP":  syscall.SIGHUP,
	"INT":  syscall.SIGINT,
	"KILL": syscall.SIGKILL,
	"QUIT": syscall.SIGQUIT,
	"TERM": syscall.SIGTERM,
	"ABRT": syscall.SIGABRT,

	"SIGHUP":  syscall.SIGHUP,
	"SIGINT":  syscall.SIGINT,
	"SIGKILL": syscall.SIGKILL,
	"SIGQUIT": syscall.SIGQUIT,
	"SIGTERM": syscall.SIGTERM,
	"SIGABRT": syscall.SIGABRT,
}

// GetSignal takes a signal name like "TERM" or "SIGTERM" and returns the syscall.Signal for it, or a zero
// syscall.Signal and false if the name does not match a known/supported signal.
func GetSignal(s string) (syscall.Signal, bool) {
	sig, ok := stopSignals[s]
	return sig, ok
}

// commandFactory creates an *exec.Cmd with platform dependant settings. This only exists to split platform specific
// code such as SysProcAttr's fields in separate files with build tags and avoid Goland marking a lot of stuff in red.
// This project depends on "unix-only" libraries anyway.
type commandFactory func(path string, args ...string) *exec.Cmd

var factory commandFactory

// Command represents the command to start a Service. All fields can be changed at any time and will have effect
// on next call to Start.
type Command struct {
	// Path is the path to the program's executable
	Path string

	// Args are the arguments passed to the program's executable.
	Args []string

	// Workdir specifies the working directory for the program.
	Workdir string

	// Env specifies the environment for the program.
	Env []string

	// StopSignal specifies the signal used to stop the program, when Stop is called on the handle returned by
	// a call to Start. If it's empty when Start is called, SIGTERM is used.
	StopSignal syscall.Signal

	// GracePeriod specifies the max time to wait for the program to quit on its own before it is
	// killed after a call to Stop on the handle returned by the call to Start.
	GracePeriod time.Duration
}

// Start starts the command using the given writers as its stdout and stderr. An error with a nil Handle is returned
// if the command fails to start. If the command starts, the Handle can be used to monitor the command lifecycle.
func (c *Command) Start(stdout, stderr io.Writer) (*handle, error) {
	if c.Path == "" {
		return nil, fmt.Errorf("empty command path")
	}

	if c.StopSignal == 0 {
		c.StopSignal = syscall.SIGTERM
	}

	cmd := factory(c.Path, c.Args...)

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = c.Workdir
	cmd.Env = append(os.Environ(), c.Env...)

	writeMessage(stderr, fmt.Sprintf("starting %q with args %+q in working dir %q", c.Path, c.Args, c.Workdir))

	if err := cmd.Start(); err != nil {
		err := fmt.Errorf("command: %w", err)
		writeMessage(stderr, err.Error())
		return nil, err
	}

	return newHandle(cmd, stderr, c.StopSignal, c.GracePeriod), nil
}

// handle represents a running program.
type handle struct {
	cmd *exec.Cmd

	err    error
	doneCh chan struct{}

	gracePeriod time.Duration
	stopSignal  os.Signal
}

// newHandle creates a new *handle for the given - already started - *exec.Cmd.
func newHandle(cmd *exec.Cmd, stderr io.Writer, stopSignal os.Signal, gracePeriod time.Duration) *handle {
	doneCh := make(chan struct{})
	h := &handle{cmd: cmd, gracePeriod: gracePeriod, doneCh: doneCh, stopSignal: stopSignal}

	go func() {
		if err := cmd.Wait(); err != nil {
			writeMessage(stderr, err.Error())
			h.err = err
		} else {
			writeMessage(stderr, "exit code 0")
		}

		close(doneCh)
	}()

	return h
}

// Wait blocks until the program finishes, returning an non-nil error if the program terminates with a
// non-zero exit code.
func (h *handle) Wait() error {
	<-h.doneCh
	return h.err
}

// Stop sends the signal set as Command.StopSignal before the call to Command.Start() (SIGTERM by default), and later
// SIGKILL if the program does not terminate before Command.GracePeriod expires.
func (h *handle) Stop() {
	if h.gracePeriod == 0 || h.stopSignal == syscall.SIGKILL {
		_ = h.cmd.Process.Signal(syscall.SIGKILL)
		return
	}

	go func() {
		select {
		case <-h.doneCh:
			return
		case <-time.After(h.gracePeriod):
			_ = h.cmd.Process.Signal(syscall.SIGKILL)
		}
	}()

	_ = h.cmd.Process.Signal(h.stopSignal)
}

func writeMessage(dst io.Writer, msg string) {
	_, _ = dst.Write([]byte(fmt.Sprintln(msg)))
}
