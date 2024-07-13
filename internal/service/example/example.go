package example

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	apierrors "github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"net/url"
)

// Example is a really dumb example for some business logic.
type Example interface {
	ObtainNextValue(ctx context.Context, minValue int64) (int64, error)
	ProvideStartValue(ctx context.Context, value int64) error
}

func New() Example {
	return &impl{
		value: 100,
	}
}

type impl struct {
	value int64
}

func (i *impl) ObtainNextValue(ctx context.Context, minValue int64) (int64, error) {
	aulogging.Info(ctx, "obtaining next value")

	i.value++
	if i.value < minValue {
		return 0, apierrors.NewConflict(ctx, apierrors.ValueTooLow, url.Values{"minimum": []string{"the current value is too low"}})
	}

	return i.value, nil
}

func (i *impl) ProvideStartValue(ctx context.Context, value int64) error {
	if value > 100 {
		return apierrors.NewBadRequest(ctx, apierrors.ValueTooHigh, url.Values{"details": []string{"the value must be less than 100"}})
	}
	i.value = value
	return nil
}
