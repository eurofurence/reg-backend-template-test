package server

import (
	"context"
	"fmt"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

type Server interface {
	Serve(handler http.Handler) error
	Shutdown() error
}

type Options struct {
	BaseCtx context.Context

	Host        string
	Port        int
	MetricsPort int

	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	ShutdownWait time.Duration
}

type server struct {
	options Options

	srv        *http.Server
	metricsSrv *http.Server

	interrupt chan os.Signal
	shutdown  chan struct{}
}

var _ Server = (*server)(nil)

func NewServer(options Options) Server {
	s := new(server)

	s.interrupt = make(chan os.Signal, 1)
	s.shutdown = make(chan struct{})

	s.options = options

	return s
}

func (s *server) Serve(handler http.Handler) error {
	s.srv = s.newServer(handler, s.options.Port)

	if s.options.MetricsPort > 0 {
		go s.serveMetricsAsync(s.options.MetricsPort)
	}

	s.setupSignalHandler()
	go s.handleInterrupt()

	aulogging.Logger.NoCtx().Info().Printf("serving requests on %s...", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-s.shutdown

	return nil
}

func (s *server) serveMetricsAsync(port int) {
	metricsServeMux := http.NewServeMux()
	metricsServeMux.Handle("/metrics", promhttp.Handler())

	s.metricsSrv = s.newServer(metricsServeMux, port)

	aulogging.Logger.NoCtx().Info().Printf("serving metrics requests on %s...", s.metricsSrv.Addr)
	if err := s.metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		aulogging.Logger.NoCtx().Error().Printf("failed to start metrics service on %s...", s.metricsSrv.Addr)
		return
	}
}

func (s *server) newServer(handler http.Handler, port int) *http.Server {
	return &http.Server{
		BaseContext: func(l net.Listener) context.Context {
			return s.options.BaseCtx
		},
		Handler:      handler,
		IdleTimeout:  s.options.IdleTimeout,
		ReadTimeout:  s.options.ReadTimeout,
		WriteTimeout: s.options.WriteTimeout,
		Addr:         fmt.Sprintf("%s:%d", s.options.Host, port),
	}
}

func (s *server) setupSignalHandler() {
	s.interrupt = make(chan os.Signal)
	signal.Notify(s.interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func (s *server) handleInterrupt() {
	<-s.interrupt
	if err := s.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

func (s *server) Shutdown() error {
	defer close(s.shutdown)

	aulogging.Logger.NoCtx().Info().Print("gracefully shutting down server")

	tCtx, cancel := context.WithTimeout(s.options.BaseCtx, s.options.ShutdownWait)
	defer cancel()

	if err := s.srv.Shutdown(tCtx); err != nil {
		return errors.Wrap(err, "couldn't gracefully shut down server")
	}
	if s.options.MetricsPort > 0 {
		if err := s.metricsSrv.Shutdown(tCtx); err != nil {
			return errors.Wrap(err, "couldn't gracefully shut down metrics server")
		}
	}

	return nil
}
