package server

import (
	"context"
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-backend-template-test/internal/application/middleware"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/idp"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

func Router(ctx context.Context, idpClient idp.IdentityProviderClient) (chi.Router, error) {
	router := chi.NewMux()

	err := setupMiddlewareStack(ctx, router, idpClient)
	if err != nil {
		return nil, err
	}

	return router, nil
}

func Serve(ctx context.Context, handler http.Handler) error {
	options := Options{
		BaseCtx:      ctx,
		Host:         auconfigenv.Get(ConfServerAddress),
		Port:         aToPort(auconfigenv.Get(ConfServerPort), 8080),
		MetricsPort:  aToPort(auconfigenv.Get(ConfMetricsPort), 0),
		IdleTimeout:  aToSeconds(auconfigenv.Get(ConfServerIdleTimeoutSeconds)),
		ReadTimeout:  aToSeconds(auconfigenv.Get(ConfServerReadTimeoutSeconds)),
		WriteTimeout: aToSeconds(auconfigenv.Get(ConfServerWriteTimeoutSeconds)),
		ShutdownWait: aToSeconds(auconfigenv.Get(ConfServerShutdownGraceSeconds)),
	}

	srv := NewServer(options)

	err := srv.Serve(handler)
	if err != nil {
		aulogging.ErrorErrf(ctx, err, "failure during serve phase - shutting down: %s", err.Error())
		return err
	}

	return nil
}

func aToPort(s string, fallback int) int {
	port, err := auconfigenv.AToInt(s)
	if err != nil {
		// config was validated so should only happen in tests, but use sensible value
		return fallback
	}
	return port
}

func aToSeconds(s string) time.Duration {
	secs, err := auconfigenv.AToInt(s)
	if err != nil {
		// config was validated so should only happen in tests, but use sensible value
		secs = 30
	}
	return time.Duration(secs) * time.Second
}

const (
	ConfServerAddress              = "SERVER_ADDRESS"
	ConfServerPort                 = "SERVER_PORT"
	ConfMetricsPort                = "METRICS_PORT"
	ConfServerIdleTimeoutSeconds   = "SERVER_IDLE_TIMEOUT_SECONDS"
	ConfServerReadTimeoutSeconds   = "SERVER_READ_TIMEOUT_SECONDS"
	ConfServerWriteTimeoutSeconds  = "SERVER_WRITE_TIMEOUT_SECONDS"
	ConfServerShutdownGraceSeconds = "SERVER_SHUTDOWN_GRACE_SECONDS"
	ConfRequestTimeoutSeconds      = "REQUEST_TIMEOUT_SECONDS"
)

func ConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         ConfServerAddress,
			Default:     "",
			Description: "ip address or hostname to listen on, can be left blank for localhost",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         ConfServerPort,
			Default:     "8080",
			Description: "port to listen on, defaults to 8080 if not set",
			Validate:    auconfigenv.ObtainUintRangeValidator(1024, 65535),
		}, {
			Key:         ConfMetricsPort,
			Default:     "9090",
			Description: "port to provide prometheus metrics on, cannot be a privileged port.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1024, 65535),
		}, {
			Key:         ConfServerIdleTimeoutSeconds,
			Default:     "60",
			Description: "request processing timeout in seconds while connection is idle.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		}, {
			Key:         ConfServerReadTimeoutSeconds,
			Default:     "30",
			Description: "request processing timeout in seconds during read.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		}, {
			Key:         ConfServerWriteTimeoutSeconds,
			Default:     "30",
			Description: "request processing timeout in seconds while writing response.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		}, {
			Key:         ConfServerShutdownGraceSeconds,
			Default:     "3",
			Description: "grace period in seconds for requests to finish processing while a graceful shutdown is under way.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		}, {
			Key:         ConfRequestTimeoutSeconds,
			Default:     "25",
			Description: "request processing timeout in seconds. Allows for a proper error response to be sent, so should be set lower than the server timeouts in most common cases.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		},
	}
}

func setupMiddlewareStack(ctx context.Context, router chi.Router, idpClient idp.IdentityProviderClient) error {
	router.Use(middleware.RequestID)

	router.Use(middleware.AddRequestScopedLoggerToContext)
	router.Use(middleware.RequestLogger)

	router.Use(middleware.PanicRecoverer)

	corsOptions := middleware.CorsOptionsFromConfig()
	router.Use(middleware.CorsHeaders(&corsOptions))

	router.Use(middleware.RequestMetrics())

	securityOptions := middleware.SecurityOptionsPartialFromConfig()
	securityOptions.IDPClient = idpClient
	router.Use(middleware.CheckRequestAuthorization(&securityOptions))

	router.Use(middleware.Timeout(aToSeconds(auconfigenv.Get(ConfRequestTimeoutSeconds))))

	return nil
}
