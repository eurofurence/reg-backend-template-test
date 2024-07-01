package middleware

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestCheckAllAuthentication(t *testing.T) {
	origCtx := context.WithValue(context.Background(), "ctxkey", "ctxval")
	checkOrigCtx := func(ctx context.Context) bool {
		return ctx.Value("ctxkey") == "ctxval"
	}
	compareErr := func(msg string) func(error) bool {
		return func(err error) bool {
			return err.Error() == msg
		}
	}

	configNoIDP := SecurityOptions{
		OpenEndpoints: []string{
			"POST a/b/open",
			"PUT open/a/b",
		},
		ApiKey:           "api-key",
		IDPClient:        nil,
		AllowedAudiences: nil,
		RequiredScopes:   nil,
	}

	testcases := []struct {
		name       string
		method     string
		urlPath    string
		conf       *SecurityOptions
		apiToken   string
		authHeader string
		expectCtx  func(context.Context) bool
		expectMsg  string
		expectErr  func(error) bool
	}{
		{
			name:       "no_idp_none_provided",
			method:     http.MethodGet,
			urlPath:    "a/b/c",
			conf:       &configNoIDP,
			apiToken:   "",
			authHeader: "",
			expectCtx:  checkOrigCtx,
			expectMsg:  "you must be logged in for this operation",
			expectErr:  compareErr("no authorization presented"),
		},
		// TODO more test cases with mocked idp client now
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, msg, err := checkAllAuthentication(origCtx, tc.method, tc.urlPath, tc.conf, tc.apiToken, tc.authHeader)
			require.True(t, tc.expectCtx(ctx))
			require.Equal(t, tc.expectMsg, msg)
			require.True(t, tc.expectErr(err))
		})
	}
}

func TestListsIntersect(t *testing.T) {
	testcases := []struct {
		name     string
		first    []string
		second   []string
		expected bool
	}{
		{
			name:     "both_nil",
			first:    nil,
			second:   nil,
			expected: false,
		},
		{
			name:     "first_empty",
			first:    []string{},
			second:   []string{"a", "b"},
			expected: false,
		},
		{
			name:     "second_empty",
			first:    []string{"a", "b"},
			second:   []string{},
			expected: false,
		},
		{
			name:     "identical",
			first:    []string{"d"},
			second:   []string{"d"},
			expected: true,
		},
		{
			name:     "intersect",
			first:    []string{"a", "b", "c", "d"},
			second:   []string{"d", "e", "f", "g"},
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, listsIntersect(tc.first, tc.second))
		})
	}
}

func TestListsContained(t *testing.T) {
	testcases := []struct {
		name     string
		haystack []string
		needles  []string
		expected bool
	}{
		{
			name:     "both_nil",
			haystack: nil,
			needles:  nil,
			expected: true,
		},
		{
			name:     "needles_empty",
			haystack: []string{"a", "b", "c"},
			needles:  []string{},
			expected: true,
		},
		{
			name:     "not_contained",
			haystack: []string{"a", "b"},
			needles:  []string{"a", "d"},
			expected: false,
		},
		{
			name:     "contained",
			haystack: []string{"a", "b", "c", "d"},
			needles:  []string{"a", "d"},
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, listsContained(tc.haystack, tc.needles))
		})
	}
}

func TestListContains(t *testing.T) {
	testcases := []struct {
		name     string
		haystack []string
		needle   string
		expected bool
	}{
		{
			name:     "nil_stack",
			haystack: nil,
			needle:   "a",
			expected: false,
		},
		{
			name:     "empty_stack",
			haystack: []string{},
			needle:   "a",
			expected: false,
		},
		{
			name:     "single_stack",
			haystack: []string{"a"},
			needle:   "a",
			expected: true,
		},
		{
			name:     "multi_stack",
			haystack: []string{"b", "a", "d"},
			needle:   "a",
			expected: true,
		},
		{
			name:     "not_contained",
			haystack: []string{"b", "f", "x"},
			needle:   "a",
			expected: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, listContains(tc.haystack, tc.needle))
		})
	}
}
