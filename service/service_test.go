package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestDaemon_StartStop(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := &Command{Path: "sleep", Args: []string{"60"}}

	d := NewDaemon("test", cmd, stdout, stderr)

	handle, err := d.Start()

	if err != nil {
		t.Fatalf("Start(): got = %v, want nil error", err)
	}

	handle.Stop()
	err = handle.Wait()
	if _, ok := err.(*exec.Error); ok {
		t.Fatalf("Wait(): got %v, want *exec.ExitError", err)
	}
}

func TestManager_StartStop(t *testing.T) {
	mgr := NewManager()
	s1 := &fakeService{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscription := mgr.Subscribe(ctx)

	mgr.Start(s1)
	<-subscription

	if got, want := mgr.Running(s1), true; got != want {
		t.Fatalf("Running(): got = %v, want = %v", got, want)
	}

	mgr.Stop(s1)
	<-subscription

	if got, want := mgr.Running(s1), false; got != want {
		t.Fatalf("Running(): got = %v, want = %v", got, want)
	}

	if got, want := s1.starts, int32(1); got != want {
		t.Fatalf("Start(): got Service.Start() called %d times, want Service.Start() called %d times", got, want)
	}
}

func TestManager_Start_AfterShutdown(t *testing.T) {
	s1 := &fakeService{name: "s1"}

	mgr := NewManager()
	mgr.Shutdown()

	mgr.Start(s1)

	if got, want := s1.starts, int32(0); got != want {
		t.Fatalf("Start() = %d, want %d effective calls to Service.Start", got, want)
	}
}

func TestNewManager_Running_AfterShutdown(t *testing.T) {
	s1 := &fakeService{name: "s1"}

	mgr := NewManager()
	mgr.Start(s1)

	mgr.Shutdown()

	if got, want := mgr.Running(s1), false; got != want {
		t.Fatalf("Running() = %t, want %t", got, want)
	}
}

func TestManager_Subscribe_Cancellation(t *testing.T) {
	s1 := &fakeService{name: "s1"}

	mgr := NewManager()

	ctx, cancel := context.WithCancel(context.Background())

	subscription1 := mgr.Subscribe(ctx)
	subscription2 := mgr.Subscribe(ctx)

	mgr.Start(s1)

	// wait until service actually started, then cancel the subscription
	// and check that at most one notification - will be 0 or 1 depending
	// on timing between context cancelation and stopping the service after it -
	// is read from the channel before it is closed.
	<-subscription1
	<-subscription2
	cancel()

	mgr.Stop(s1)

	assertReadAtMost(subscription1, 1, t)
	assertReadAtMost(subscription2, 1, t)
}

func TestManager_Subscribe_Shutdown(t *testing.T) {
	s1 := &fakeService{name: "s1"}

	mgr := NewManager()

	ctx := context.Background()

	subscription1 := mgr.Subscribe(ctx)
	subscription2 := mgr.Subscribe(ctx)
	mgr.Start(s1)

	// wait until service actually started, then shutdown
	// and check that at most one - depending on timing -
	// notification (caused by the service being stopped)
	// is read before the channel is closed.
	<-subscription1
	<-subscription2
	mgr.Shutdown()

	assertReadAtMost(subscription1, 1, t)
	assertReadAtMost(subscription2, 1, t)
}

func TestManager_Subscribe_AfterShutdown(t *testing.T) {
	s1 := &fakeService{name: "s1"}

	mgr := NewManager()
	mgr.Start(s1)
	mgr.Shutdown()

	ctx := context.Background()
	subscription1 := mgr.Subscribe(ctx)
	subscription2 := mgr.Subscribe(ctx)

	mgr.Shutdown()
	assertReadAtMost(subscription1, 0, t)
	assertReadAtMost(subscription2, 0, t)
}

func assertReadAtMost(ch <-chan Change, atMost int, t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var counter int

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: timed out while waiting for messages", t.Name())
		case _, ok := <-ch:
			if !ok {
				break loop
			}

			counter ++
			if counter > atMost {
				t.Fatalf("%s: read %d messages, want at most %d", t.Name(), counter, atMost)
			}
		}
	}
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
	name      string
	starts    int32
	stopDelay time.Duration
}

func (f *fakeService) Name() string {
	return fmt.Sprintf("fakeService[%q]", f.name)
}

func (f *fakeService) String() string {
	return f.Name()
}

func (f *fakeService) Start() (Handle, error) {
	atomic.AddInt32(&f.starts, 1)
	return &fakeHandle{done: make(chan struct{}), stopDelay: f.stopDelay}, nil
}
