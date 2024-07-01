package common

import (
	"context"
	"github.com/eurofurence/reg-backend-template-test/internal/apimodel"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/timestamp"
	"net/http"
	"net/url"
)

// APIError allows lower layers of the service to provide detailed information about an error.
//
// While this breaks layer separation somewhat, it avoids having to map errors all over the place.
type APIError interface {
	error
	Status() int
	Response() apimodel.Error
}

// ErrorMessageCode is a key to use for error messages in frontends or other automated systems interacting
// with our API. It avoids having to parse human-readable language for error classification beyond the
// http status.
type ErrorMessageCode string

const (
	AuthUnauthorized     ErrorMessageCode = "auth.unauthorized" // token missing completely or invalid or expired
	AuthForbidden        ErrorMessageCode = "auth.forbidden"    // permissions missing
	RequestParseFailed   ErrorMessageCode = "request.parse.failed"
	ValueTooHigh         ErrorMessageCode = "value.too.high"
	ValueTooLow          ErrorMessageCode = "value.too.low"
	InternalErrorMessage ErrorMessageCode = "error.internal"
	UnknownErrorMessage  ErrorMessageCode = "error.unknown"
)

// construct specific API errors

func NewBadRequest(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusBadRequest, message, details)
}

func NewUnauthorized(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusUnauthorized, message, details)
}

func NewForbidden(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusForbidden, message, details)
}

func NewNotFound(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusNotFound, message, details)
}

func NewConflict(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusConflict, message, details)
}

func NewInternalServerError(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusInternalServerError, message, details)
}

func NewBadGateway(ctx context.Context, message ErrorMessageCode, details url.Values) APIError {
	return NewAPIError(ctx, http.StatusBadGateway, message, details)
}

// check for API errors

func IsBadRequestError(err error) bool {
	return isAPIErrorWithStatus(http.StatusBadRequest, err)
}

func IsUnauthorizedError(err error) bool {
	return isAPIErrorWithStatus(http.StatusUnauthorized, err)
}

func IsForbiddenError(err error) bool {
	return isAPIErrorWithStatus(http.StatusForbidden, err)
}

func IsNotFoundError(err error) bool {
	return isAPIErrorWithStatus(http.StatusNotFound, err)
}

func IsConflictError(err error) bool {
	return isAPIErrorWithStatus(http.StatusConflict, err)
}

func IsBadGatewayError(err error) bool {
	return isAPIErrorWithStatus(http.StatusBadGateway, err)
}

func IsInternalServerError(err error) bool {
	return isAPIErrorWithStatus(http.StatusInternalServerError, err)
}

func IsAPIError(err error) bool {
	_, ok := err.(APIError)
	return ok
}

// NewAPIError creates a generic API error from directly provided information.
func NewAPIError(ctx context.Context, status int, message ErrorMessageCode, details url.Values) APIError {
	return &StatusError{
		errStatus: status,
		response: apimodel.Error{
			Timestamp: timestamp.Now(),
			Requestid: GetRequestID(ctx),
			Message:   string(message),
			Details:   details,
		},
	}
}

var _ error = (*StatusError)(nil)

type StatusError struct {
	errStatus int
	response  apimodel.Error
}

func (se *StatusError) Error() string {
	return se.response.Message
}

func (se *StatusError) Status() int {
	return se.errStatus
}

func (se *StatusError) Response() apimodel.Error {
	return se.response
}

func isAPIErrorWithStatus(status int, err error) bool {
	apiError, ok := err.(APIError)
	return ok && status == apiError.Status()
}
