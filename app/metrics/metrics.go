package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mrz.io/hkswitch/service"
	"net"
	"net/http"
	"time"
)

const serviceStateMetricName = "hkswitch_up"

var serviceStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: serviceStateMetricName}, []string{"service"})

// ConsumeServiceStateChanges updates the hkswitch_up metric from the given subscription channel.
func ConsumeServiceStateChanges(subscription <-chan service.Change) {
	go func() {
		for change := range subscription {
			updateServiceState(change.Service.Name(), change.Running)
		}
	}()
}

// updateServiceState records a new value for the up/down state of the named service.
func updateServiceState(name string, up bool) {
	var v float64

	if up {
		v = 1.0
	}

	serviceStateMetric.WithLabelValues(name).Set(v)
}

type Server struct {
	srv *http.Server
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	return &Server{srv: srv}, nil
}
