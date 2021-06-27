package homekit

import (
	"errors"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"mrz.io/hkswitch/app/config"
	"mrz.io/hkswitch/service"
	"time"
)

// Bridge exposes Services to HomeKit.
type Bridge struct {
	cfg config.Config

	mgr *service.Manager

	transport hc.Transport

	startStopCh chan bool
	startErr    error
	didStopCh   chan error
}

// NewBridge creates a new Bridge that uses the service.Manager to control the given services. A nil Bridge and error
// can be returned if initializing the underlying hc.Transport fails.
func NewBridge(cfg config.Config, mgr *service.Manager, services ...service.Service) (*Bridge, error) {
	b := &Bridge{}

	b.cfg = cfg
	b.mgr = mgr
	b.didStopCh = make(chan error)
	b.startStopCh = make(chan bool)

	t, err := b.initializeTransport(services)

	if err != nil {
		return nil, err
	}

	b.transport = t

	go func() {
		var started bool
		var didStopCh <-chan error

		for {
			select {
			case err := <-didStopCh:
				started = false
				b.didStopCh <- err
				return
			case newState := <-b.startStopCh:
				if newState == started {
					continue
				}

				if newState {
					didStopCh = b.startTransport()
					started = true
				} else {
					b.stopTransport()
				}
			}
		}

	}()

	return b, nil
}

// Start starts the bridge (including underlying transport), blocking until a call to Stop.
func (b *Bridge) Start() error {
	select {
	case <-b.startStopCh:
		return nil
	default:
		b.startStopCh <- true
		return <-b.didStopCh
	}
}

func (b *Bridge) startTransport() <-chan error {
	didStopCh := make(chan error)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				didStopCh <- errors.New(r.(string))
			}
			close(didStopCh)
		}()

		b.transport.Start()
	}()

	return didStopCh
}

// Stop stops the bridge and the underlying transport, blocking until a call to Stop.
func (b *Bridge) Stop() {
	select {
	case <-b.startStopCh:
	default:
		b.startStopCh <- false
		<-b.didStopCh
	}
}

func (b *Bridge) stopTransport() {
	go func() {
		<-b.transport.Stop()
	}()
}

func (b *Bridge) initializeTransport(services []service.Service) (hc.Transport, error) {
	bridgeInfo := accessory.Info{
		Name:             b.cfg.Name,
		Manufacturer:     b.cfg.Manufacturer,
		SerialNumber:     b.cfg.SerialNumber,
		Model:            b.cfg.Model,
		FirmwareRevision: b.cfg.Firmware,
	}
	bridge := accessory.NewBridge(bridgeInfo)

	switches, accessories := b.createSwitches(services)

	transportConfig := hc.Config{Pin: b.cfg.Pin, Port: b.cfg.Port, StoragePath: b.cfg.StorageDir}
	t, err := hc.NewIPTransport(transportConfig, bridge.Accessory, accessories...)
	if err != nil {
		return nil, err
	}

	b.startStopServicesBySwitch(services, switches)
	b.updateSwitchByServiceState(switches, services)

	return t, nil
}

func (b *Bridge) createSwitches(services []service.Service) ([]*accessory.Switch, []*accessory.Accessory) {
	var switches []*accessory.Switch
	var accessories []*accessory.Accessory

	for _, s := range services {
		info := accessory.Info{Name: s.Name()}
		switchAcc := accessory.NewSwitch(info)

		switches = append(switches, switchAcc)
		accessories = append(accessories, switchAcc.Accessory)
	}

	return switches, accessories
}

func (b *Bridge) startStopServicesBySwitch(services []service.Service, switches []*accessory.Switch) {
	for i, svc := range services {
		svc := svc
		acc := switches[i]

		acc.Switch.On.OnValueRemoteUpdate(func(on bool) {
			if on {
				b.mgr.Start(svc)
			} else {
				b.mgr.Stop(svc)
			}
		})
	}
}

func (b *Bridge) updateSwitchByServiceState(switches []*accessory.Switch, services []service.Service) {
	go func() {
		for {
			<-time.After(1 * time.Second)
			for i, acc := range switches {
				acc.Switch.On.SetValue(b.mgr.Running(services[i]))
			}
		}
	}()
}
