package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/eurofurence/reg-backend-template-test/internal/application/web"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/idp"
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"
)

type SecurityOptions struct {
	// OpenEndpoints is the list of endpoints that can be accessed without requiring a valid token or api key
	//
	// Any endpoint not in this list will reject access.
	//
	// Even if an endpoint is in this list, if a token or api key is present, they are still validated to catch
	// expired or retracted credentials. The endpoint may behave differently if authorization is presented, after all.
	//
	// Format: METHOD PATH
	// Example: GET /
	OpenEndpoints []string

	// ApiKey is a fixed shared secret. It uses a separate header so this can be filtered in an ingress.
	ApiKey string

	// OpenID Connect, these may be extracted from the .well-known endpoint during middleware setup
	IDPClient idp.IdentityProviderClient

	AllowedAudiences []string
	RequiredScopes   []string
}

const (
	ConfOIDCAllowedAudiences = "OIDC_ALLOWED_AUDIENCES"
	ConfOIDCRequiredScopes   = "OIDC_REQUIRED_SCOPES"
	ConfApiKey               = "API_KEY"
	ConfOpenEndpoints        = "OPEN_ENDPOINTS"
)

func SecurityConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         ConfOIDCAllowedAudiences,
			Default:     "",
			Description: "Audiences of the token to allow through. A token is authorized if its audience is in this list. Will need a token introspection endpoint to be present in OpenID Well Known response. Accepts a space separated list. If empty, all audiences are allowed through (may not be secure).",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         ConfOIDCRequiredScopes,
			Default:     "",
			Description: "Scopes of the token to allow through. A token is authorized only if it has all scopes in this list. Will need a token introspection endpoint to be present in OpenID Well Known response. Accepts a space separated list. If empty, no scopes are checked (may not be secure).",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         ConfApiKey,
			Default:     "",
			Description: "Shared secret API Key. Uses a separate header, which should be filtered in the ingress to prevent use from the outside.",
			Validate:    auconfigapi.ConfigNeedsNoValidation,
		}, {
			Key:         ConfOpenEndpoints,
			Default:     `["GET /", "GET /favicon.ico"]`,
			Description: "List of endpoints which can be called without authorization.",
			Validate:    validateOpenEndpoints,
		},
	}
}

func parseOpenEndpoints(value string) ([]string, error) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	result := make([]string, 0)
	err := decoder.Decode(&result)
	if err != nil {
		return result, fmt.Errorf("failed to parse open endpoint configuration: %w", err)
	}
	return result, nil
}

func validateOpenEndpoints(key string) error {
	val := auconfigenv.Get(key)
	_, err := parseOpenEndpoints(val)
	return err
}

func SecurityOptionsPartialFromConfig() SecurityOptions {
	openEndpoints, _ := parseOpenEndpoints(auconfigenv.Get(ConfOpenEndpoints))
	return SecurityOptions{
		ApiKey:           auconfigenv.Get(ConfApiKey),
		AllowedAudiences: splitBySpaceOrEmpty(auconfigenv.Get(ConfOIDCAllowedAudiences)),
		RequiredScopes:   splitBySpaceOrEmpty(auconfigenv.Get(ConfOIDCRequiredScopes)),
		OpenEndpoints:    openEndpoints,
	}
}

func splitBySpaceOrEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, " ")
}

const (
	apiKeyHeader = "X-Api-Key"
	bearerPrefix = "Bearer "
)

// CheckRequestAuthorization creates a middleware that validates authorization and adds them to the relevant
// context values.
func CheckRequestAuthorization(conf *SecurityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			apiTokenHeaderValue := fromApiTokenHeader(r)
			authHeaderValue := fromAuthHeader(r)

			ctx, userFacingErrorMessage, err := checkAllAuthentication(ctx, r.Method, r.URL.Path, conf, apiTokenHeaderValue, authHeaderValue)
			if err != nil {
				subject := common.GetSubject(ctx)
				aulogging.InfoErrf(ctx, err, "authorization failed for subject %s: %s", subject, userFacingErrorMessage)
				web.SendUnauthorizedResponse(ctx, w, userFacingErrorMessage)
				return
			}

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// internals

// --- getting the values from the request ---

func fromAuthHeader(r *http.Request) string {
	headerValue := r.Header.Get(headers.Authorization)

	if !strings.HasPrefix(headerValue, bearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(headerValue, bearerPrefix)
}

func fromApiTokenHeader(r *http.Request) string {
	return r.Header.Get(apiKeyHeader)
}

// --- top level ---.

func checkAllAuthentication(ctx context.Context, method string, urlPath string, conf *SecurityOptions, apiTokenHeaderValue string, authHeaderValue string) (context.Context, string, error) {
	var success bool
	var err error

	// try api token first
	ctx, success, err = checkApiToken(ctx, conf, apiTokenHeaderValue)
	if err != nil {
		return ctx, "invalid api token", err
	}
	if success {
		return ctx, "", nil
	}

	// now try authorization header (gives only access token, so MUST use userinfo/tokeninfo endpoint)
	ctx, success, err = checkAccessToken(ctx, conf, authHeaderValue)
	if err != nil {
		return ctx, "invalid bearer token", err
	}
	if success {
		return ctx, "", nil
	}

	// allow through (but still AFTER auth processing)
	currentEndpoint := fmt.Sprintf("%s %s", method, urlPath)
	for _, publicEndpoint := range conf.OpenEndpoints {
		if currentEndpoint == publicEndpoint {
			return ctx, "", nil
		}
	}

	return ctx, "you must be logged in for this operation", errors.New("no authorization presented")
}

// --- validating the individual pieces ---

// important - if any of these return an error, you must abort processing via "return" and log the error message

func checkApiToken(ctx context.Context, conf *SecurityOptions, apiTokenValue string) (context.Context, bool, error) {
	if apiTokenValue != "" {
		// ignore jwt if set (may still need to pass it through to other service)
		if apiTokenValue == conf.ApiKey {
			ctx = context.WithValue(ctx, common.CtxKeyAPIKey{}, apiTokenValue)
			return ctx, true, nil
		} else {
			return ctx, false, errors.New("token doesn't match the configured value")
		}
	}
	return ctx, false, nil
}

func checkAccessToken(ctx context.Context, conf *SecurityOptions, accessTokenValue string) (context.Context, bool, error) {
	if accessTokenValue != "" {
		if conf.IDPClient != nil {
			authCtx := context.WithValue(ctx, common.CtxKeyAccessToken{}, accessTokenValue) // need this set for userinfo call

			userInfo, status, err := conf.IDPClient.UserInfo(authCtx)
			if err != nil {
				return ctx, false, fmt.Errorf("request failed access token check, denying: %s", err.Error())
			}
			if status != http.StatusOK {
				return ctx, false, fmt.Errorf("request failed access token check with status %d, denying", status)
			}

			if len(conf.AllowedAudiences) > 0 {
				if !listsIntersect(conf.AllowedAudiences, userInfo.Audience) {
					return ctx, false, errors.New("token audience does not contain a match")
				}
			}

			if len(conf.RequiredScopes) > 0 {
				tokenInfo, status, err := conf.IDPClient.TokenIntrospection(ctx)
				if err != nil {
					return ctx, false, fmt.Errorf("request failed access token introspection, denying: %s", err.Error())
				}
				if status != http.StatusOK {
					return ctx, false, fmt.Errorf("request failed access token introspection with status %d, denying", status)
				}

				if !listsContained(strings.Split(tokenInfo.Scope, " "), conf.RequiredScopes) {
					return ctx, false, errors.New("token does not have all required scopes")
				}
			}

			overwriteClaims := common.AllClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:   conf.IDPClient.Issuer(),
					Subject:  userInfo.Subject,
					Audience: jwt.ClaimStrings(userInfo.Audience),
				},
				CustomClaims: common.CustomClaims{
					EMail:         userInfo.Email,
					EMailVerified: userInfo.EmailVerified,
					Groups:        userInfo.Groups,
					Name:          userInfo.Name,
				},
			}

			ctx = context.WithValue(authCtx, common.CtxKeyClaims{}, &overwriteClaims)
			return ctx, true, nil
		} else {
			return ctx, false, errors.New("request failed access token check, denying: no userinfo endpoint configured")
		}
	}
	return ctx, false, nil
}

// listsIntersect is true if at least one common element exists.
//
// If either list is empty, they do not intersect.
func listsIntersect(firstList []string, secondList []string) bool {
	for _, first := range firstList {
		for _, second := range secondList {
			if first == second {
				return true
			}
		}
	}
	return false
}

// listsContained is true if all needles are in the haystack.
//
// An empty list of needles is considered contained in any haystack.
func listsContained(haystack []string, needles []string) bool {
	for _, needle := range needles {
		if !listContains(haystack, needle) {
			return false
		}
	}
	return true
}

func listContains(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if needle == candidate {
			return true
		}
	}
	return false
}
