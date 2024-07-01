package middleware

import (
	"context"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

var RequestIDHeader = "X-Request-Id"

var ValidRequestIdRegex = regexp.MustCompile("^[0-9a-f]{8}$")

// RequestID obtains the request id from the request header, or failing that, creates a new request id,
// and places it in the request context.
//
// It also adds it to the response under the same header.
//
// This automatically also leads to all logging using this context to log the request id.
func RequestID(next http.Handler) http.Handler {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		reqUuidStr := r.Header.Get(RequestIDHeader)
		if !ValidRequestIdRegex.MatchString(reqUuidStr) {
			reqUuid, err := uuid.NewRandom()
			if err == nil {
				reqUuidStr = reqUuid.String()[:8]
			} else {
				// this should not normally ever happen, but continue with this fixed requestId
				reqUuidStr = "ffffffff"
			}
		}
		ctx := r.Context()
		newCtx := context.WithValue(ctx, common.CtxKeyRequestID{}, reqUuidStr)
		r = r.WithContext(newCtx)
		w.Header().Add(RequestIDHeader, reqUuidStr)

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handlerFunc)
}
