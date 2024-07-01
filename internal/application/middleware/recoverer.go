package middleware

import (
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"net/http"
	"runtime/debug"
)

func PanicRecoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rvr := recover()
			if rvr != nil && rvr != http.ErrAbortHandler {
				ctx := r.Context()
				stack := string(debug.Stack())
				aulogging.Error(ctx, "recovered from PANIC: "+stack)
				web.SendErrorWithStatusAndMessage(ctx, w, http.StatusInternalServerError, common.InternalErrorMessage, "")
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
