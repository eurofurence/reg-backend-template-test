package main

import (
	"github.com/eurofurence/reg-backend-template-test/internal/logging"
	"github.com/eurofurence/reg-backend-template-test/internal/server"

	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// TODO start implementing your service here

	logging.NoCtx().Info("Service is starting")

	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
		logging.NoCtx().Info("Stopping service now")

		tCtx, tcancel := context.WithTimeout(context.Background(), time.Second*5)
		defer tcancel()

		if err := server.Shutdown(tCtx); err != nil {
			logging.NoCtx().Fatal("Couldn't shutdown server gracefully")
		}
	}()

	handler := server.Create()
	server.Serve(ctx, handler)
}
