package idpmock

import (
	"context"
	"errors"
	"github.com/eurofurence/reg-backend-template-test/internal/application/common"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/idp"
	"net/http"
)

func New() idp.IdentityProviderClient {
	return &impl{
		userInfo:   make(map[string]idp.UserinfoResponse),
		tokenIntro: make(map[string]idp.TokenIntrospectionResponse),
	}
}

type impl struct {
	userInfo   map[string]idp.UserinfoResponse
	tokenIntro map[string]idp.TokenIntrospectionResponse
	issuer     string
}

func (i *impl) SetupFromWellKnown(ctx context.Context) error {
	i.issuer = "mock-issuer"
	return nil
}

func (i *impl) Issuer() string {
	return i.issuer
}

func (i *impl) UserInfo(ctx context.Context) (*idp.UserinfoResponse, int, error) {
	token := common.GetAccessToken(ctx)

	uInf, ok := i.userInfo[token]
	if !ok {
		return nil, http.StatusUnauthorized, errors.New("unknown token")
	}

	return &uInf, http.StatusOK, nil
}

func (i *impl) TokenIntrospection(ctx context.Context) (*idp.TokenIntrospectionResponse, int, error) {
	token := common.GetAccessToken(ctx)

	tInt, ok := i.tokenIntro[token]
	if !ok {
		return nil, http.StatusUnauthorized, errors.New("unknown token")
	}

	return &tInt, http.StatusOK, nil
}

func SetupResponse(instance idp.IdentityProviderClient, token string, userInfo idp.UserinfoResponse, tokenIntro idp.TokenIntrospectionResponse) {
	mock, ok := instance.(*impl)
	if ok {
		mock.userInfo[token] = userInfo
		mock.tokenIntro[token] = tokenIntro
	}
}
