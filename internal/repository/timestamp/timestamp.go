package timestamp

import "time"

var nowFunc = time.Now

func Now() time.Time {
	return nowFunc()
}

// exposed for testing

func SetFakeNow(fakeNow time.Time) {
	nowFunc = func() time.Time {
		return fakeNow
	}
}
