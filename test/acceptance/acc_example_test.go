package acceptance

import (
	"github.com/eurofurence/reg-backend-template-test/docs"
	"github.com/eurofurence/reg-backend-template-test/internal/apimodel"
	"net/http"
	"testing"
)

// ------------------------------------------
// acceptance tests for the example resource
// ------------------------------------------

func TestExample_Success(t *testing.T) {
	tstSetup(t)
	defer tstShutdown()

	docs.Given("given a logged in regular user")
	token := tstValidUserToken(t, 101)
	tstSetupIDPResponse(t, 101, nil)

	docs.When("when they request the example resource")
	response := tstPerformGet("/api/rest/v1/example", token)

	docs.Then("then a valid response is sent with the next value")
	tstRequireSuccessResponse(t, response, http.StatusOK, &apimodel.Example{Value: 42})
}

// security tests

func TestExample_DenyUnauthorized(t *testing.T) {
	tstSetup(t)
	defer tstShutdown()

	docs.Given("given an anonymous user")

	docs.When("when they request the example resource")
	response := tstPerformGet("/api/rest/v1/example", tstNoToken())

	docs.Then("then the request is denied as unauthorized (401)")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")
}
