package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
)

const (
	deployerNotEnoughCalls = "event manager didn't have the right number of calls"
)

var _ = Describe("Controller", func() {

	var (
		controller   *Controller
		deployer     *mocks.Deployer
		eventManager *mocks.EventManager
		fetcher      *mocks.Fetcher
		context      *gin.Context
		resp         *httptest.ResponseRecorder
		router       *gin.Engine

		environment     string
		org             string
		space           string
		appName         string
		defaultUsername string
		defaultPassword string
		apiURL          string

		jsonBuffer *bytes.Buffer
	)

	BeforeEach(func() {
		deployer = &mocks.Deployer{}
		eventManager = &mocks.EventManager{}
		fetcher = &mocks.Fetcher{}

		jsonBuffer = &bytes.Buffer{}

		envMap := map[string]config.Environment{}
		envMap["Test"] = config.Environment{Foundations: []string{"api1.example.com", "api2.example.com"}}
		envMap["Prod"] = config.Environment{Foundations: []string{"api3.example.com", "api4.example.com"}}

		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		defaultUsername = "defaultUsername-" + randomizer.StringRunes(10)
		defaultPassword = "defaultPassword-" + randomizer.StringRunes(10)

		c := config.Config{
			Username:     defaultUsername,
			Password:     defaultPassword,
			Environments: envMap,
		}

		controller = &Controller{
			Config:       c,
			Deployer:     deployer,
			Log:          logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
			EventManager: eventManager,
			Fetcher:      fetcher,
		}

		apiURL = fmt.Sprintf("/v1/apps/%s/%s/%s/%s",
			environment,
			org,
			space,
			appName,
		)

		context, resp, router = gin.CreateTestContext()

		router.POST("/v1/apps/:environment/:org/:space/:appName", controller.Deploy)

		deployer.DeployCall.Received.EnvironmentName = environment
		deployer.DeployCall.Received.Org = org
		deployer.DeployCall.Received.Space = space
		deployer.DeployCall.Received.AppName = appName
		deployer.DeployCall.Received.Out = jsonBuffer
	})

	Describe("type application/json", func() {
		Context("without missing properties", func() {
			It("deploys successfully with a status code of 200 OK", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = 200

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body).To(ContainSubstring("deploy successful"))
			})
		})

		Context("when an application fails", func() {
			It("returns an error", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				req.Header.Set("Content-Type", "application/json")

				deployer.DeployCall.Returns.Error = errors.New("internal server error")
				deployer.DeployCall.Returns.StatusCode = 500

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("internal server error"))
			})
		})
	})

	Describe("type application/zip", func() {
		Context("with a valid zip file", func() {
			It("deploys successfully with a status code of 200 OK", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				fetcher.FetchFromZipCall.Received.RequestBody = nil
				fetcher.FetchFromZipCall.Returns.AppPath = "appPath-" + randomizer.StringRunes(10)
				fetcher.FetchFromZipCall.Returns.Error = nil

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body).To(ContainSubstring("deploy successful"))
			})
		})

		Context("when the request is empty", func() {
			It("returns an error", func() {
				req, err := http.NewRequest("POST", apiURL, nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(0), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(400))
				Expect(resp.Body).To(ContainSubstring("request body is empty"))
			})
		})

		Context("when it cannot process the zip file", func() {
			It("returns an error", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				fetcher.FetchFromZipCall.Received.RequestBody = jsonBuffer.Bytes()
				fetcher.FetchFromZipCall.Returns.AppPath = ""
				fetcher.FetchFromZipCall.Returns.Error = errors.New("could not process zip file")

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(0), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("could not process zip file"))
			})
		})

		Context("when an application fails", func() {
			It("returns an error", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				fetcher.FetchFromZipCall.Received.RequestBody = jsonBuffer.Bytes()
				fetcher.FetchFromZipCall.Returns.AppPath = "appPath-" + randomizer.StringRunes(10)
				fetcher.FetchFromZipCall.Returns.Error = nil

				deployer.DeployCall.Returns.Error = errors.New("internal server error")
				deployer.DeployCall.Returns.StatusCode = 500

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1), deployerNotEnoughCalls)
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("cannot deploy application"))
			})
		})
	})

	Describe("unknown content type", func() {
		It("returns an error", func() {
			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "invalidContentType")

			deployer.DeployCall.Returns.Error = errors.New("internal server error")
			deployer.DeployCall.Returns.StatusCode = 400

			router.ServeHTTP(resp, req)

			Expect(deployer.DeployCall.TimesCalled).To(Equal(0), deployerNotEnoughCalls)
			Expect(deployer.DeployCall.TimesCalled).To(Equal(0), deployerNotEnoughCalls)
			Expect(resp.Code).To(Equal(400))
			Expect(resp.Body).To(ContainSubstring("content type not supported - must be application/json or application/zip"))
		})
	})
})
