package acceptance

import (
	"fmt"
	"github.com/eurofurence/reg-backend-template-test/internal/repository/idp"
	"github.com/eurofurence/reg-backend-template-test/test/mocks/idpmock"
	"testing"
)

func tstNoToken() string {
	return ""
}

func tstValidUserToken(t *testing.T, id uint) string {
	t.Helper()

	return fmt.Sprintf("valid_user_token_%d", id)
}

func tstSetupIDPResponse(t *testing.T, id uint, groups []string) {
	t.Helper()

	token := tstValidUserToken(t, id)
	aud := []string{"14d9f37a-1eec-47c9-a949-5f1ebdf9c8e5"}
	sub := fmt.Sprintf("%d", id)

	idpmock.SetupResponse(application.IDPClient, token, idp.UserinfoResponse{
		Audience:      aud,
		AuthTime:      1516239022,
		Email:         "demouser@example.com",
		EmailVerified: true,
		Name:          "John Doe",
		Groups:        groups,
		Issuer:        "http://identity.localhost/",
		IssuedAt:      1516239022,
		RequestedAt:   1516239022,
		Subject:       sub,
	}, idp.TokenIntrospectionResponse{
		Active:    true,
		Scope:     "groups fun",
		ClientId:  "1a4f",
		Sub:       sub,
		Exp:       2075120816,
		Iat:       1516239022,
		Aud:       aud,
		Iss:       "http://identity.localhost/",
		TokenType: "",
		TokenUse:  "",
	})
}
