package service

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestDaemon_Start_Stop(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := &Command{Path: "sleep",Args: []string{"60"}}

	d := NewDaemon("test", cmd, stdout, stderr)

	handle, err := d.Start()

	if err != nil {
		t.Fatalf("is = %v, want = nil", err)
	}

	handle.Stop()
	_ = handle.Wait()
}

type fakeStreamFactory struct {
}

func (f fakeStreamFactory) Stdout(svc Service) io.Writer {
	return &bytes.Buffer{}
}

func (f fakeStreamFactory) Stderr(svc Service) io.Writer {
	return &bytes.Buffer{}
}

type fakeHandle struct {
	done      chan struct{}
	stopDelay time.Duration
}

func (f *fakeHandle) Wait() error {
	<-f.done
	<-time.After(f.stopDelay)
	return nil
}

func (f *fakeHandle) Stop() {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
}

type fakeService struct {
	starts    int32
	stopDelay time.Duration
}

func (f *fakeService) Name() string {
	return "test"
}

func (f *fakeService) String() string {
	return f.Name()
}

func (f *fakeService) Start() (Handle, error) {
	atomic.AddInt32(&f.starts, 1)
	return &fakeHandle{done: make(chan struct{}), stopDelay: f.stopDelay}, nil
}

func TestNewManager(t *testing.T) {
	mgr := NewManager(nil)

	s1 := &fakeService{stopDelay: 1 * time.Second}
	s2 := &fakeService{}

	mgr.Start(s1, s2, s1)

	if is, want := mgr.Running(s1), true; is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}

	if is, want := mgr.Running(s2), true; is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}

	mgr.Stop(s2)

	<-time.After(100 * time.Millisecond)

	mgr.Shutdown()

	if is, want := mgr.Running(s1), false; is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}

	if is, want := mgr.Running(s2), false; is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}

	if is, want := s1.starts, int32(1); is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}

	if is, want := s2.starts, int32(1); is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}
}
