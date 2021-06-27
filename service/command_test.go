package service

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

func badprog(t *testing.T) string {
	f, err := ioutil.TempFile("", "")

	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "build", "-o", f.Name(), "./testdata/badprog.go")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	return f.Name()
}

func TestHandle_Wait(t *testing.T) {
	stderr := &bytes.Buffer{}

	cmd := exec.Command("bash", "-c", "echo hello")
	if is, want := cmd.Start(), error(nil); is != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}

	handle := newHandle(cmd, stderr, syscall.SIGTERM, 5*time.Second)

	if is, want := handle.Wait(), error(nil); is != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}
}

func TestHandle_Wait_WithError(t *testing.T) {
	stderr := &bytes.Buffer{}

	cmd := exec.Command("bash", "-c", "exit 42")
	if is, want := cmd.Start(), error(nil); is != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}

	handle := newHandle(cmd, stderr, syscall.SIGTERM, 5*time.Second)

	if is, want := handle.Wait(), "exit status 42"; is.Error() != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}
}

func TestHandle_Stop_WithKill(t *testing.T) {
	t.SkipNow()
	stdoutr, stdoutw, _ := os.Pipe()

	stderr := &bytes.Buffer{}

	cmd := exec.Command(badprog(t))
	cmd.Stdout = stdoutw

	if is, want := cmd.Start(), error(nil); is != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}

	handle := newHandle(cmd, stderr, syscall.SIGTERM, 100*time.Millisecond)
	go func() {
		scanner := bufio.NewScanner(stdoutr)

		for scanner.Scan() {
			handle.Stop()
			return
		}
	}()

	if is, want := handle.Wait(), "signal: killed"; is.Error() != want {
		t.Fatalf("is=%v, want=%v", is, want)
	}
}

func TestCommand_Start_Stop(t *testing.T) {
	cmd := &Command{
		Path:        "sleep",
		Args:        []string{"10"},
		GracePeriod: 5 * time.Second,
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	handle, err := cmd.Start(stdout, stderr)

	if err != nil {
		t.Fatalf("is = %v, want = %v", err, nil)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if is, want := handle.Wait(), "signal: terminated"; is.Error() != want {
			t.Errorf("is = %v, want = %v", is, want)
		}
	}()

	handle.Stop()
	wg.Wait()
}

func TestCommand_Start_WithError(t *testing.T) {
	cmd := &Command{Path: "foo"}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	handle, err := cmd.Start(stdout, stderr)
	want := "command: exec: \"foo\": executable file not found in $PATH"
	if err.Error() != want {
		t.Fatalf("is = %v, want = %v", err, want)
	}

	if handle != nil {
		t.Fatalf("is = %v, want = nil", handle)
	}

}
