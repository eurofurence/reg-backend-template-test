package middleware

import (
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/go-http-utils/headers"
	"strings"

	"net/http"
)

type CorsOptions struct {
	Enable        bool
	AllowOrigin   string
	ExposeHeaders []string // Location, X-Request-Id, ...
}

func CorsHeaders(options *CorsOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if options != nil && options.Enable {
				aulogging.Infof(ctx, "sending headers to disable CORS from %s", options.AllowOrigin)
				w.Header().Set(headers.AccessControlAllowOrigin, options.AllowOrigin)
				w.Header().Set(headers.AccessControlAllowMethods, "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set(headers.AccessControlAllowHeaders, "content-type")
				w.Header().Set(headers.AccessControlAllowCredentials, "true")
				w.Header().Set(headers.AccessControlExposeHeaders, strings.Join(options.ExposeHeaders, ", "))
			}

			if r.Method == http.MethodOptions {
				aulogging.Info(ctx, "received OPTIONS request. Responding with OK.")
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

const (
	ConfCorsHeadersEnable = "CORS_HEADERS_ENABLE"
	ConfCorsAllowOrigin   = "CORS_ALLOW_ORIGIN"
)

func CorsConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         ConfCorsHeadersEnable,
			Default:     "0",
			Description: "enable CORS headers. Must also configure CORS_ALLOW_ORIGIN or this will not work. Set to '1' to enable.",
			Validate:    auconfigenv.ObtainUintRangeValidator(0, 1),
		}, {
			Key:         ConfCorsAllowOrigin,
			Default:     "",
			Description: "origin to allow via CORS headers. With recent browsers, '*' no longer works. Example: 'http://localhost:8000'.",
			Validate:    auconfigenv.ObtainPatternValidator("^(|https?://.*)$"),
		},
	}
}

func CorsOptionsFromConfig() CorsOptions {
	return CorsOptions{
		Enable:      auconfigenv.Get(ConfCorsHeadersEnable) == "1",
		AllowOrigin: auconfigenv.Get(ConfCorsAllowOrigin),
		ExposeHeaders: []string{
			"Location",
			RequestIDHeader,
		},
	}
}
