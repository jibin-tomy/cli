package serviceauthtoken_test

import (
	. "cf/commands/serviceauthtoken"
	"cf/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
)

func callListServiceAuthTokens(reqFactory *testreq.FakeReqFactory, authTokenRepo *testapi.FakeAuthTokenRepo) (ui *testterm.FakeUI) {
	ui = &testterm.FakeUI{}

	config := testconfig.NewRepositoryWithDefaults()

	cmd := NewListServiceAuthTokens(ui, config, authTokenRepo)
	ctxt := testcmd.NewContext("service-auth-tokens", []string{})
	testcmd.RunCommand(cmd, ctxt, reqFactory)

	return
}

var _ = Describe("Testing with ginkgo", func() {
	It("TestListServiceAuthTokensRequirements", func() {
		authTokenRepo := &testapi.FakeAuthTokenRepo{}
		reqFactory := &testreq.FakeReqFactory{}

		reqFactory.LoginSuccess = false
		callListServiceAuthTokens(reqFactory, authTokenRepo)
		Expect(testcmd.CommandDidPassRequirements).To(BeFalse())

		reqFactory.LoginSuccess = true
		callListServiceAuthTokens(reqFactory, authTokenRepo)
		Expect(testcmd.CommandDidPassRequirements).To(BeTrue())
	})
	It("TestListServiceAuthTokens", func() {

		reqFactory := &testreq.FakeReqFactory{LoginSuccess: true}
		authTokenRepo := &testapi.FakeAuthTokenRepo{}
		authToken := models.ServiceAuthTokenFields{}
		authToken.Label = "a label"
		authToken.Provider = "a provider"
		authToken2 := models.ServiceAuthTokenFields{}
		authToken2.Label = "a second label"
		authToken2.Provider = "a second provider"
		authTokenRepo.FindAllAuthTokens = []models.ServiceAuthTokenFields{authToken, authToken2}

		ui := callListServiceAuthTokens(reqFactory, authTokenRepo)
		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Getting service auth tokens as", "my-user"},
			{"OK"},
			{"label", "provider"},
			{"a label", "a provider"},
			{"a second label", "a second provider"},
		})
	})
})
