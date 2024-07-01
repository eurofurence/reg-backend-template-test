package idp

import (
	"context"
	auconfigapi "github.com/StephanHCB/go-autumn-config-api"
	auconfigenv "github.com/StephanHCB/go-autumn-config-env"
)

type IdentityProviderClient interface {
	// SetupFromWellKnown must be called at least once before any other methods can be used.
	SetupFromWellKnown(ctx context.Context) error

	Issuer() string

	// UserInfo extracts the token from the context and performs a user info lookup
	UserInfo(ctx context.Context) (*UserinfoResponse, int, error)

	// TokenIntrospection extracts the token from the context and performs a token info lookup
	TokenIntrospection(ctx context.Context) (*TokenIntrospectionResponse, int, error)
}

const (
	ConfOIDCWellKnownURL         = "OIDC_WELL_KNOWN_URL"
	ConfTokenIntrospectionURL    = "IDP_TOKEN_INTROSPECTION_URL"
	ConfIDPRequestTimeoutSeconds = "IDP_REQUEST_TIMEOUT_SECONDS"
	ConfIDPCacheEnabled          = "IDP_CACHE_ENABLED"
	ConfIDPCacheRetentionSeconds = "IDP_CACHE_RETENTION_SECONDS"
)

func ConfigItems() []auconfigapi.ConfigItem {
	return []auconfigapi.ConfigItem{
		{
			Key:         ConfOIDCWellKnownURL,
			Default:     "",
			Description: "URL of the OpenID Connect .well-known endpoint. If set, allows Identity Provider integration with autoconfiguration.",
			Validate:    auconfigenv.ObtainPatternValidator("^(|https?://.*)$"),
		},
		{
			Key:         ConfTokenIntrospectionURL,
			Default:     "",
			Description: "URL of the token introspection endpoint. If set, allows Identity Provider to validate scopes.",
			Validate:    auconfigenv.ObtainPatternValidator("^(|https?://.*)$"),
		}, {
			Key:         ConfIDPRequestTimeoutSeconds,
			Default:     "10",
			Description: "timeout in seconds for all requests to the identity provider.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 1800),
		}, {
			Key:         ConfIDPCacheEnabled,
			Default:     "0",
			Description: "cache IDP responses. Off by default. Enable by setting this to '1', but be aware of the security implications, namely that revoked tokens will still be accepted for the cache duration. Tradeoff with performance.",
			Validate:    auconfigenv.ObtainUintRangeValidator(0, 1),
		}, {
			Key:         ConfIDPCacheRetentionSeconds,
			Default:     "5",
			Description: "cache IDP responses for this many seconds. Please be aware of the security implications, namely that revoked tokens will still be accepted for this many seconds. Tradeoff with performance. Limited to 30 seconds, defaults to 5. Note that the cache is off by default, so this setting only has an effect if IDP_CACHE_ENABLED is set to 1.",
			Validate:    auconfigenv.ObtainUintRangeValidator(1, 30),
		},
	}
}

type UserinfoResponse struct {
	// can leave out fields - we are using a tolerant reader
	Audience      []string `json:"aud"`
	AuthTime      int64    `json:"auth_time"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"` // username
	Groups        []string `json:"groups"`
	Issuer        string   `json:"iss"`
	IssuedAt      int64    `json:"iat"`
	RequestedAt   int64    `json:"rat"`
	Subject       string   `json:"sub"`

	// in case of error, you get these fields instead
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type TokenIntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope"`
	ClientId  string   `json:"client_id"`
	Sub       string   `json:"sub"`
	Exp       int64    `json:"exp"`
	Iat       int64    `json:"iat"`
	Nbf       int64    `json:"nbf"`
	Aud       []string `json:"aud"`
	Iss       string   `json:"iss"`
	TokenType string   `json:"token_type"`
	TokenUse  string   `json:"token_use"`

	// in case of error, you get these fields instead
	ErrorMessage string              `json:"message"`
	Errors       map[string][]string `json:"errors"`
}

type WellKnownResponse struct {
	Issuer           string `json:"issuer"`
	UserinfoEndpoint string `json:"userinfo_endpoint"`
}
