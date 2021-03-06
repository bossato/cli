package api_test

import (
	"net/http"

	"github.com/cloudfoundry/cli/cf/api/apifakes"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/net"
	testassert "github.com/cloudfoundry/cli/testhelpers/assert"
	"github.com/cloudfoundry/cli/testhelpers/cloudcontrollergateway"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testnet "github.com/cloudfoundry/cli/testhelpers/net"

	. "github.com/cloudfoundry/cli/cf/api"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CloudControllerCurlRepository ", func() {
	var (
		headers string
		body    string
		apiErr  error
	)

	Describe("GET requests", func() {
		BeforeEach(func() {
			req := apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
				Method: "GET",
				Path:   "/v2/endpoint",
				Response: testnet.TestResponse{
					Status: http.StatusOK,
					Body:   expectedJSONResponse},
			})
			ts, handler := testnet.NewServer([]testnet.TestRequest{req})
			defer ts.Close()

			deps := newCurlDependencies()
			deps.config.SetAPIEndpoint(ts.URL)

			repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
			headers, body, apiErr = repo.Request("GET", "/v2/endpoint", "", "")

			Expect(handler).To(HaveAllRequestsCalled())
			Expect(apiErr).NotTo(HaveOccurred())
		})

		It("returns headers with the status code", func() {
			Expect(headers).To(ContainSubstring("200"))
		})

		It("returns the header content type", func() {
			Expect(headers).To(ContainSubstring("Content-Type"))
			Expect(headers).To(ContainSubstring("text/plain"))
		})

		It("returns the body as a JSON-encoded string", func() {
			testassert.JSONStringEquals(body, expectedJSONResponse)
		})
	})

	Describe("POST requests", func() {
		BeforeEach(func() {
			req := apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
				Method:  "POST",
				Path:    "/v2/endpoint",
				Matcher: testnet.RequestBodyMatcher(`{"key":"val"}`),
				Response: testnet.TestResponse{
					Status: http.StatusOK,
					Body:   expectedJSONResponse},
			})

			ts, handler := testnet.NewServer([]testnet.TestRequest{req})
			defer ts.Close()

			deps := newCurlDependencies()
			deps.config.SetAPIEndpoint(ts.URL)

			repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
			headers, body, apiErr = repo.Request("POST", "/v2/endpoint", "", `{"key":"val"}`)
			Expect(handler).To(HaveAllRequestsCalled())
		})

		It("does not return an error", func() {
			Expect(apiErr).NotTo(HaveOccurred())
		})

		Context("when the server returns a 400 Bad Request header", func() {
			BeforeEach(func() {
				req := apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
					Method:  "POST",
					Path:    "/v2/endpoint",
					Matcher: testnet.RequestBodyMatcher(`{"key":"val"}`),
					Response: testnet.TestResponse{
						Status: http.StatusBadRequest,
						Body:   expectedJSONResponse},
				})

				ts, handler := testnet.NewServer([]testnet.TestRequest{req})
				defer ts.Close()

				deps := newCurlDependencies()
				deps.config.SetAPIEndpoint(ts.URL)

				repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
				_, body, apiErr = repo.Request("POST", "/v2/endpoint", "", `{"key":"val"}`)
				Expect(handler).To(HaveAllRequestsCalled())
			})

			It("returns the response body", func() {
				testassert.JSONStringEquals(body, expectedJSONResponse)
			})

			It("does not return an error", func() {
				Expect(apiErr).NotTo(HaveOccurred())
			})
		})

		Context("when provided with invalid headers", func() {
			It("returns an error", func() {
				deps := newCurlDependencies()
				repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
				_, _, apiErr := repo.Request("POST", "/v2/endpoint", "not-valid", "")
				Expect(apiErr).To(HaveOccurred())
				Expect(apiErr.Error()).To(ContainSubstring("headers"))
			})
		})

		Context("when provided with valid headers", func() {
			It("sends them along with the POST body", func() {
				req := apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "POST",
					Path:   "/v2/endpoint",
					Matcher: func(req *http.Request) {
						Expect(req.Header.Get("content-type")).To(Equal("ascii/cats"))
						Expect(req.Header.Get("x-something-else")).To(Equal("5"))
					},
					Response: testnet.TestResponse{
						Status: http.StatusOK,
						Body:   expectedJSONResponse},
				})
				ts, handler := testnet.NewServer([]testnet.TestRequest{req})
				defer ts.Close()

				deps := newCurlDependencies()
				deps.config.SetAPIEndpoint(ts.URL)

				headers := "content-type: ascii/cats\nx-something-else:5"
				repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
				_, _, apiErr := repo.Request("POST", "/v2/endpoint", headers, "")
				Expect(handler).To(HaveAllRequestsCalled())
				Expect(apiErr).NotTo(HaveOccurred())
			})
		})
	})

	It("uses POST as the default method when a body is provided", func() {
		ccServer := ghttp.NewServer()
		ccServer.AppendHandlers(
			ghttp.VerifyRequest("POST", "/v2/endpoint"),
		)

		deps := newCurlDependencies()
		deps.config.SetAPIEndpoint(ccServer.URL())

		repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
		_, _, err := repo.Request("", "/v2/endpoint", "", "body")
		Expect(err).NotTo(HaveOccurred())

		Expect(ccServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("uses GET as the default method when a body is not provided", func() {
		ccServer := ghttp.NewServer()
		ccServer.AppendHandlers(
			ghttp.VerifyRequest("GET", "/v2/endpoint"),
		)

		deps := newCurlDependencies()
		deps.config.SetAPIEndpoint(ccServer.URL())

		repo := NewCloudControllerCurlRepository(deps.config, deps.gateway)
		_, _, err := repo.Request("", "/v2/endpoint", "", "")
		Expect(err).NotTo(HaveOccurred())

		Expect(ccServer.ReceivedRequests()).To(HaveLen(1))
	})
})

const expectedJSONResponse = `
	{"resources": [
		{
		  "metadata": { "guid": "my-quota-guid" },
		  "entity": { "name": "my-remote-quota", "memory_limit": 1024 }
		}
	]}
`

type curlDependencies struct {
	config  coreconfig.ReadWriter
	gateway net.Gateway
}

func newCurlDependencies() (deps curlDependencies) {
	deps.config = testconfig.NewRepository()
	deps.config.SetAccessToken("BEARER my_access_token")
	deps.gateway = cloudcontrollergateway.NewTestCloudControllerGateway(deps.config)
	return
}
