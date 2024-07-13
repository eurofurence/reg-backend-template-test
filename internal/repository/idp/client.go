package idp

import (
	"context"
	"errors"
	"fmt"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestbreakerprometheus "github.com/StephanHCB/go-autumn-restclient-circuitbreaker-prometheus"
	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientprometheus "github.com/StephanHCB/go-autumn-restclient-prometheus"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestcaching "github.com/StephanHCB/go-autumn-restclient/implementation/caching"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/go-http-utils/headers"
	"net/http"
	"time"
)

type Options struct {
	RequestTimeout time.Duration

	CacheEnabled       bool
	CacheRetentionTime time.Duration

	OIDCWellKnownURL string

	TokenIntrospectionURL string
}

// --- instance creation ---

func New(options Options) IdentityProviderClient {
	instance := Impl{
		options: options,
	}

	httpClient, err := auresthttpclient.New(0, nil, instance.requestManipulator)
	if err != nil {
		aulogging.Logger.NoCtx().Fatal().WithErr(err).Printf("Failed to instantiate IDP client - BAILING OUT: %s", err.Error())
	}
	aurestclientprometheus.InstrumentHttpClient(httpClient)

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		"identity-provider-breaker",
		10,
		2*time.Minute,
		30*time.Second,
		options.RequestTimeout,
	)
	aurestbreakerprometheus.InstrumentCircuitBreakerClient(circuitBreakerClient)

	client := circuitBreakerClient

	if options.CacheEnabled {
		cachingClient := aurestcaching.New(circuitBreakerClient,
			instance.useCacheCondition,
			storeResponseCondition,
			cacheKeyFunction,
			options.CacheRetentionTime,
			256,
		)
		aurestclientprometheus.InstrumentCacheClient(cachingClient)
		client = cachingClient
	}

	instance.client = client

	return &instance
}

func OptionsFromConfig() Options {
	return Options{
		RequestTimeout:        aToSeconds(auconfigenv.Get(ConfIDPRequestTimeoutSeconds)),
		CacheEnabled:          auconfigenv.Get(ConfIDPCacheEnabled) == "1",
		CacheRetentionTime:    aToSeconds(auconfigenv.Get(ConfIDPCacheRetentionSeconds)),
		OIDCWellKnownURL:      auconfigenv.Get(ConfOIDCWellKnownURL),
		TokenIntrospectionURL: auconfigenv.Get(ConfTokenIntrospectionURL),
	}
}

func aToSeconds(s string) time.Duration {
	secs, err := auconfigenv.AToInt(s)
	if err != nil {
		// config was validated so should only happen in tests, but use sensible value
		secs = 10
	}
	return time.Duration(secs) * time.Second
}

type Impl struct {
	client  aurestclientapi.Client
	options Options

	oidcUserInfoURL string
	issuer          string
}

// useCacheCondition determines whether the cache should be used for a given request
//
// we cache only GET requests to the configured userinfo endpoint, and only for users who present a valid auth token
func (i *Impl) useCacheCondition(ctx context.Context, method string, url string, requestBody interface{}) bool {
	return method == http.MethodGet && url == i.oidcUserInfoURL && common.GetAccessToken(ctx) != ""
}

// storeResponseCondition determines whether to store a response in the cache
//
// we only cache responses of successful requests to the userinfo endpoint
func storeResponseCondition(ctx context.Context, method string, url string, requestBody interface{}, response *aurestclientapi.ParsedResponse) bool {
	return response.Status == http.StatusOK
}

// cacheKeyFunction determines the key to cache the response under
//
// we cannot use the default cache key function, we must cache per token
func cacheKeyFunction(ctx context.Context, method string, requestUrl string, requestBody interface{}) string {
	return fmt.Sprintf("%s %s %s", common.GetAccessToken(ctx), method, requestUrl)
}

// requestManipulator inserts Authorization when we are calling the userinfo endpoint
func (i *Impl) requestManipulator(ctx context.Context, r *http.Request) {
	if r.Method == http.MethodGet {
		urlStr := r.URL.String()
		if urlStr != "" && (urlStr == i.oidcUserInfoURL) {
			r.Header.Set(headers.Authorization, "Bearer "+common.GetAccessToken(ctx))
		}
	}
}

// --- implementation of IdentityProviderClient interface ---

func (i *Impl) SetupFromWellKnown(ctx context.Context) error {
	bodyDto := WellKnownResponse{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, i.options.OIDCWellKnownURL, nil, &response)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("error requesting well known info from identity provider: error is %s", err.Error())
		return err
	}

	if bodyDto.Issuer == "" || bodyDto.UserinfoEndpoint == "" {
		aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("error requesting well known info from identity provider: no issuer or user info endpoint set - probably not a valid .well-known endpoint")
		return errors.New("failed to obtain issuer or user info endpoint from .well-known endpoint")
	}

	i.issuer = bodyDto.Issuer
	i.oidcUserInfoURL = bodyDto.UserinfoEndpoint

	return nil
}

func (i *Impl) Issuer() string {
	return i.issuer
}

func (i *Impl) UserInfo(ctx context.Context) (*UserinfoResponse, int, error) {
	userinfoEndpoint := i.oidcUserInfoURL
	bodyDto := UserinfoResponse{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, userinfoEndpoint, nil, &response)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("error requesting user info from identity provider: error from response is %s:%s, local error is %s", bodyDto.ErrorCode, bodyDto.ErrorDescription, err.Error())
		return nil, http.StatusBadGateway, err
	}
	if bodyDto.ErrorCode != "" || bodyDto.ErrorDescription != "" {
		aulogging.Logger.Ctx(ctx).Error().Printf("received an error response from identity provider: error from response is %s:%s", bodyDto.ErrorCode, bodyDto.ErrorDescription)
	}
	if response.Status != http.StatusOK && response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
		err = fmt.Errorf("unexpected http status %d, was expecting 200, 401, or 403", response.Status)
		aulogging.Logger.Ctx(ctx).Error().Printf("error requesting user info from identity provider: error from response is %s:%s, local error is %s", bodyDto.ErrorCode, bodyDto.ErrorDescription, err.Error())
		return nil, response.Status, err
	}
	if response.Status == http.StatusOK {
		if bodyDto.ErrorCode != "" || bodyDto.ErrorDescription != "" {
			err = fmt.Errorf("received an error response from identity provider: error from response is %s:%s", bodyDto.ErrorCode, bodyDto.ErrorDescription)
			return nil, response.Status, err
		}
	}

	if bodyDto.Subject != "" {
		// got old response
		return &bodyDto, response.Status, nil
	}

	return &bodyDto, response.Status, nil
}

func (i *Impl) TokenIntrospection(ctx context.Context) (*TokenIntrospectionResponse, int, error) {
	tokenIntrospectionEndpoint := i.options.TokenIntrospectionURL
	bodyDto := TokenIntrospectionResponse{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, tokenIntrospectionEndpoint, nil, &response)

	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("error requesting user info from identity provider: error from response is %s:%v, local error is %s", bodyDto.ErrorMessage, bodyDto.Errors, err.Error())
		return nil, http.StatusBadGateway, err
	}
	if bodyDto.ErrorMessage != "" || len(bodyDto.Errors) > 0 {
		aulogging.Logger.Ctx(ctx).Error().Printf("received an error response from identity provider: error from response is %s:%v", bodyDto.ErrorMessage, bodyDto.Errors)
	}
	if response.Status != http.StatusOK && response.Status != http.StatusUnauthorized && response.Status != http.StatusForbidden {
		err = fmt.Errorf("unexpected http status %d, was expecting 200, 401, or 403", response.Status)
		aulogging.Logger.Ctx(ctx).Error().Printf("error requesting user info from identity provider: error from response is %s:%v, local error is %s", bodyDto.ErrorMessage, bodyDto.Errors, err.Error())
		return nil, response.Status, err
	}
	if response.Status == http.StatusOK {
		if bodyDto.ErrorMessage != "" || len(bodyDto.Errors) > 0 {
			err = fmt.Errorf("received an error response from identity provider: error from response is %s:%v", bodyDto.ErrorMessage, bodyDto.Errors)
			return nil, response.Status, err
		}
	}

	return &bodyDto, response.Status, nil
}
