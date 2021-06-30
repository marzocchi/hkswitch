package service

import (
	"context"
	"fmt"
	"io"
	"time"
)

type Service interface {
	// Start starts the service, and returns an Handle to represent the running Service, or a nil Handle and an error
	// if the service cannot be started. Calling Start multiple times should start multiple instances of the Service.
	Start() (Handle, error)

	Name() string

	fmt.Stringer
}

// Handle represents a running Service.
type Handle interface {
	// Wait blocks until the service stopped (either because it decided to stop on its own, or because Stop was called).
	Wait() error

	// Stop stops the service.
	Stop()
}

type Change struct {
	Service   Service
	Running   bool
	Timestamp time.Time
}

type query struct {
	svc   Service
	reply chan bool
}

func newQuery(svc Service) query {
	return query{svc: svc, reply: make(chan bool)}
}

type Callback func(svc Service, running bool)

type Manager struct {
	// shutdown is closed to signal that shutdown has to be initiated.
	shutdown chan struct{}

	// didShutdown is closed to signal that shutdown has finished.
	didShutdown chan struct{}

	// running is a map to a running Service and its Handle
	running map[Service]Handle

	// start
	start   chan Service
	stop    chan Service
	stopped chan Service

	subscribe     chan chan Change
	unsubscribe   chan chan Change
	subscriptions []chan Change

	queries chan query
}

func NewManager() *Manager {
	mgr := &Manager{
		didShutdown: make(chan struct{}),
		shutdown:    make(chan struct{}),
		start:       make(chan Service),
		stop:        make(chan Service),
		stopped:     make(chan Service),
		queries:     make(chan query),
		running:     make(map[Service]Handle),
		subscribe:   make(chan chan Change),
		unsubscribe: make(chan chan Change),
	}

	go func() {
	loop:
		for {
			select {
			case ch := <-mgr.subscribe:
				mgr.addSubscription(ch)
			case rmCh := <-mgr.unsubscribe:
				mgr.removeSubscription(rmCh)
			case svc := <-mgr.start:
				mgr.startService(svc)
			case q := <-mgr.queries:
				mgr.queryService(q)
			case svc := <-mgr.stop:
				mgr.stopService(svc)
			case svc := <-mgr.stopped:
				mgr.serviceStopped(svc)
			case <-mgr.shutdown:
				break loop
			}
		}

		mgr.performShutdown()
	}()

	return mgr
}

// Subscribe returns a new channel to which a Change is written every time a service
// starts and stops, and it's closed when ctx is canceled or the Manager shutdowns.
// There might still be Changes to read after subscription is canceled.
// If Subscribe is called after shutdown a closed channel is returned.
func (mgr *Manager) Subscribe(ctx context.Context) <-chan Change {
	ch := make(chan Change, 1000)

	select {
	case <-mgr.shutdown:
		// no one's tracking new subscription, so just return a closed chan
		// to fulfill the return type with and that's it
		close(ch)
	default:
		go func() {
			select {
			case <-mgr.shutdown:
				// we can stop here, then subscription will be deleted and the channel closed
				// as part of shutting down
				return
			case <-ctx.Done():
				mgr.unsubscribe <- ch
			}
		}()

		mgr.subscribe <- ch
	}

	return ch
}

func (mgr *Manager) addSubscription(ch chan Change) {
	mgr.subscriptions = append(mgr.subscriptions, ch)
}

func (mgr *Manager) removeSubscription(rmCh chan Change) {
	defer func() {
		close(rmCh)
	}()

	var tmp []chan Change
	for _, ch := range mgr.subscriptions {
		if ch != rmCh {
			tmp = append(tmp, ch)
		}
	}
	mgr.subscriptions = tmp
}

func (mgr *Manager) notifySubscribers(svc Service, running bool) {
	t := time.Now()
	for _, ch := range mgr.subscriptions {
		select {
		case <-mgr.shutdown:
		case ch <- Change{Service: svc, Running: running, Timestamp: t}:
		default:
		}
	}
}

// performShutdown stops any running service and waits for the services to report stopped, serving any query in
// the meanwhile.
func (mgr *Manager) performShutdown() {
	defer func() {
		for _, ch := range mgr.subscriptions {
			mgr.removeSubscription(ch)
		}

		close(mgr.didShutdown)
	}()

	if len(mgr.running) == 0 {
		return
	}

	for _, h := range mgr.running {
		h.Stop()
	}

loop:
	for {
		select {
		case q := <-mgr.queries:
			mgr.queryService(q)
		case svc := <-mgr.stopped:
			mgr.serviceStopped(svc)

			if len(mgr.running) == 0 {
				break loop
			}
		}
	}
}

// waitHandle asynchronously writes the service back to stopped when Handle.Wait returns.
func (mgr *Manager) waitHandle(handle Handle, svc Service) {
	go func() {
		_ = handle.Wait()
		mgr.stopped <- svc
	}()
}

// Start starts the Services if they are not running. It does nothing if called after shutdown was initiate by
// a call to Shutdown.
func (mgr *Manager) Start(services ...Service) {
	select {
	case <-mgr.shutdown:
	default:
		for _, svc := range services {
			mgr.start <- svc
		}
	}
}

// startService starts the Service, adding it to running only if it starts without error.
func (mgr *Manager) startService(svc Service) {
	if _, ok := mgr.running[svc]; ok {
		return
	}

	handle, err := svc.Start()
	if err != nil {
		return
	}

	mgr.running[svc] = handle
	mgr.waitHandle(handle, svc)
	mgr.notifySubscribers(svc, true)
}

// Running reports on the current running state of a service.
func (mgr *Manager) Running(svc Service) bool {
	q := newQuery(svc)

	select {
	case <-mgr.didShutdown:
		return false
	case mgr.queries <- q:
		return <-q.reply
	}
}

// queryService checks if a service is running and writes the state to query.reply.
func (mgr *Manager) queryService(q query) {
	_, ok := mgr.running[q.svc]
	q.reply <- ok
}

// Stop stops one or more services. Has no effect when called after Shutdown, as all running services will be
// in the process of stopping or already stopped.
func (mgr *Manager) Stop(services ...Service) {
	select {
	case <-mgr.shutdown:
	default:
		for _, svc := range services {
			mgr.stop <- svc
		}
	}
}

// stopService stops the services if it is running.
func (mgr *Manager) stopService(svc Service) {
	if handle, ok := mgr.running[svc]; ok {
		handle.Stop()
	}
}

// serviceStopped removes the service from running.
func (mgr *Manager) serviceStopped(svc Service) {
	if _, ok := mgr.running[svc]; ok {
		delete(mgr.running, svc)
		mgr.notifySubscribers(svc, false)
	}
}

// Shutdown initiate stopping all running services, blocking until all have stopped. Further calls return immediately.
// Once Shutdown returns, the Manager can't be used anymore to Start or Stop services.
func (mgr *Manager) Shutdown() {
	select {
	case <-mgr.shutdown:
	default:
		close(mgr.shutdown)
		<-mgr.didShutdown
	}
}

type daemon struct {
	name string
	cmd  *Command

	stdout io.Writer
	stderr io.Writer
}

func NewDaemon(name string, cmd *Command, stdout, stderr io.Writer) Service {
	return &daemon{name: name, cmd: cmd, stderr: stderr, stdout: stdout}
}

func (s *daemon) String() string {
	return s.name
}

func (s *daemon) Start() (Handle, error) {
	handle, err := s.cmd.Start(s.stdout, s.stderr)
	if err != nil {
		return nil, err
	}

	return handle, nil
}

func (s *daemon) Name() string {
	return s.name
}
