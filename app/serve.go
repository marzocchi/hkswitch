package app

import (
	"context"
	"github.com/brutella/hc/log"
	"io"
	"mrz.io/hkswitch/app/config"
	"mrz.io/hkswitch/app/metrics"
	"mrz.io/hkswitch/app/output"
	"mrz.io/hkswitch/homekit"
	"mrz.io/hkswitch/service"
	"os"
	"syscall"
	"time"
)

var Version = "v0.0.1"

const (
	Name         = "hkswitch"
	manufacturer = "mrz.io"
	separator    = " | "
)

func init() {
	config.DefaultConfig.Bridge.Manufacturer = manufacturer
	config.DefaultConfig.Bridge.Model = Name
	config.DefaultConfig.Bridge.Firmware = Version
}

type StreamsFactory interface {
	Stdout(svc config.Service) io.Writer
	Stderr(svc config.Service) io.Writer
}

func Serve(ctx context.Context, stdout, stderr io.Writer, configFile string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}

	// figure out the maximum output's left column size, based on the longest between
	// the app's name and service names.
	prefixLen := output.FindPrefixSize(len(Name), cfg.ServiceNames()...)

	// log.Debug.SetOutput(output.WithPrefix(os.Stderr, output.Prefix(Name, prefixLen, separator)))
	log.Info.SetOutput(output.WithPrefix(os.Stderr, output.Prefix(Name, prefixLen, separator)))

	if cfg.Metric.Address != "" {
		metricsServer, err := metrics.NewServer(cfg.Metric.Address)
		if err != nil {
			return err
		}

		defer func() {
			if err := metricsServer.Stop(); err != nil {
				log.Info.Printf("metrics: %s\n", err)
			}
		}()
	}

	services, err := createServices(cfg, output.NewPrefixer(stdout, stderr, prefixLen, separator))
	if err != nil {
		return err
	}

	mgr := service.NewManager()
	metrics.ConsumeServiceStateChanges(mgr.Subscribe(ctx))

	bridge, err := homekit.NewBridge(cfg, mgr, services...)
	if err != nil {
		return err
	}

	shutdownOnCtxDone(ctx, bridge, mgr)
	autostart(mgr, services, cfg)

	log.Info.Printf("starting bridge...")
	err = bridge.Start()

	return err
}

func autostart(mgr *service.Manager, services []service.Service, cfg config.Config) {
	startupServices := getStartupServices(services, cfg)
	if len(startupServices) == 0 {
		return
	}

	log.Info.Printf("startup services: %+q", startupServices)
	mgr.Start(startupServices...)
}

func shutdownOnCtxDone(ctx context.Context, bridge *homekit.Bridge, mgr *service.Manager) {
	go func() {
		<-ctx.Done()
		mgr.Shutdown()
		bridge.Stop()
	}()
}

func getStartupServices(services []service.Service, cfg config.Config) []service.Service {
	var list []service.Service
	for i, svc := range cfg.Services {
		if svc.Autostart {
			list = append(list, services[i])
		}
	}

	return list
}

func createServices(cfg config.Config, sf StreamsFactory) ([]service.Service, error) {
	var list []service.Service

	for _, svcCfg := range cfg.Services {
		sig, ok := service.GetSignal(svcCfg.StopSignal)
		if !ok {
			sig = syscall.SIGTERM
		}

		cmd := &service.Command{
			Path:        svcCfg.Command[0],
			Args:        svcCfg.Command[1:],
			Workdir:     svcCfg.Workdir,
			Env:         svcCfg.Env,
			StopSignal:  sig,
			GracePeriod: 5 * time.Second,
		}

		svc := service.NewDaemon(svcCfg.Name, cmd, sf.Stdout(svcCfg), sf.Stderr(svcCfg))

		list = append(list, svc)
	}

	return list, nil
}
