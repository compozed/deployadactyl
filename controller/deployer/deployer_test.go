package deployer_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
)

const (
	deployAborted = "Deploy aborted, one or more CF foundations unavailable"
	testManifest  = `---
applications:
- name: deployadactyl
  memory: 256M
  disk_quota: 256M
`
	testManifestFile           = "./manifest.yml"
	jsonRequest                = "application/json"
	zipRequest                 = "application/zip"
	eventManagerNotEnoughCalls = "event manager didn't have the right number of calls"
)

var _ = Describe("Deployer", func() {
	var (
		deployer Deployer

		c              config.Config
		blueGreener    *mocks.BlueGreener
		fetcher        *mocks.Fetcher
		prechecker     *mocks.Prechecker
		eventManager   *mocks.EventManager
		randomizerMock *mocks.Randomizer

		req             *http.Request
		reqBuffer       *bytes.Buffer
		appName         string
		appPath         string
		artifactURL     string
		domain          string
		environmentName string
		org             string
		space           string
		username        string
		uuid            string
		password        string
		context         *gin.Context

		deploymentInfo  S.DeploymentInfo
		deployEventData S.DeployEventData
		foundations     []string
		environments    = map[string]config.Environment{}
		log             = logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "test")
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)

		blueGreener = &mocks.BlueGreener{}
		fetcher = &mocks.Fetcher{}
		prechecker = &mocks.Prechecker{}
		eventManager = &mocks.EventManager{}
		randomizerMock = &mocks.Randomizer{}

		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		artifactURL = "artifactURL-" + randomizer.StringRunes(10)
		domain = "domain-" + randomizer.StringRunes(10)
		environmentName = "environmentName-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		uuid = "uuid-" + randomizer.StringRunes(10)

		randomizerMock.RandomizeCall.Returns.Runes = uuid

		reqBuffer = bytes.NewBufferString(fmt.Sprintf(`{
		  		"artifact_url": "%s"
		  	}`,
			artifactURL,
		))
		req, _ = http.NewRequest("POST", "", reqBuffer)

		Expect(ioutil.WriteFile(testManifestFile, []byte(testManifest), 0644)).To(Succeed())

		deploymentInfo = S.DeploymentInfo{
			ArtifactURL: artifactURL,
			Username:    username,
			Password:    password,
			Environment: environmentName,
			Org:         org,
			Space:       space,
			AppName:     appName,
			UUID:        uuid,
		}

		deployEventData = S.DeployEventData{
			Writer:         &bytes.Buffer{},
			DeploymentInfo: &deploymentInfo,
		}

		randomizerMock.RandomizeCall.Returns.Runes = uuid

		foundations = []string{randomizer.StringRunes(10)}
		context, _, _ = gin.CreateTestContext()

		environments = map[string]config.Environment{}
		environments[environmentName] = config.Environment{
			Name:        environmentName,
			Domain:      domain,
			Foundations: foundations,
		}

		c = config.Config{
			Username:     username,
			Password:     password,
			Environments: environments,
		}

		deployer = Deployer{c, blueGreener, fetcher, prechecker, eventManager, randomizerMock, log}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(testManifestFile)).To(Succeed())
	})

	Describe("deploy JSON", func() {

		Context("when fetcher fails", func() {
			It("returns an error", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				fetcher.FetchCall.Returns.Error = errors.New("Fetcher error")
				fetcher.FetchCall.Returns.AppPath = appPath

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, appPath, "application/json", context)
				Expect(err).To(MatchError("Fetcher error"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
			})
		})

		Context("with missing properties in the JSON", func() {
			It("returns an error", func() {
				By("sending empty JSON")
				reqBuffer = bytes.NewBufferString("{}")
				req, _ = http.NewRequest("POST", "", reqBuffer)

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("The following properties are missing: artifact_url"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when all applications start correctly", func() {
			It("is successful", func() {
				eventManager.EmitCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath
				fetcher.FetchCall.Returns.Error = nil
				blueGreener.PushCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(BeNil())
				Expect(statusCode).To(Equal(http.StatusOK))

				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
			})
		})

		Context("when an application fails to start", func() {
			It("returns an error", func() {
				eventManager.EmitCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				fetcher.FetchCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath

				By("making bluegreener return an error")
				blueGreener.PushCall.Returns.Error = errors.New("blue green error")

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("blue green error"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})

		Context("when eventmanager fails on deploy.start", func() {
			It("returns an error", func() {
				By("making eventmanager return an error")
				eventManager.EmitCall.Returns.Error = errors.New("event error")

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("an error occurred in the deploy.start event"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				// Expect(context).To(ContainSubstring("event error"))
			})
		})

		Context("when custom manifest information is given in the request body", func() {
			It("properly decodes base64 encoding of the provided manifest information", func() {
				eventManager.EmitCall.Returns.Error = nil

				deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

				By("base64 encoding a manifest")
				base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

				By("including manifest in the JSON")
				reqBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	 					"artifact_url": "%s",
	 					"manifest": "%s"
	 				}`,
					artifactURL,
					base64Manifest,
				))
				req, _ = http.NewRequest("POST", "", reqBuffer)

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).ToNot(HaveOccurred())

				Expect(statusCode).To(Equal(http.StatusOK))
				// Expect(context).To(ContainSubstring("deploy was successful"))
				Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
			})

			It("returns an error if the provided manifest information is not base64 encoded", func() {
				deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

				By("not base64 encoding a manifest")

				By("including manifest in the JSON")
				reqBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	 					"artifact_url": "%s",
	 					"manifest": "%s"
	 				}`,
					artifactURL,
					deploymentInfo.Manifest,
				))
				req, _ = http.NewRequest("POST", "", reqBuffer)

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("cannot open manifest file"))
				Expect(statusCode).To(Equal(http.StatusBadRequest))

				Expect(eventManager.EmitCall.TimesCalled).To(Equal(0), eventManagerNotEnoughCalls)
			})
		})
	})

	Describe("deploy zip", func() {
		Context("when all applications start correctly", func() {
			It("accepts the request with a http.StatusOK OK", func() {
				eventManager.EmitCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				blueGreener.PushCall.Returns.Error = nil

				By("getting a temporary manifest from the working directory")
				wd, err := os.Getwd()
				Expect(err).To(BeNil())

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, wd, zipRequest, context)
				Expect(err).To(BeNil())
				Expect(statusCode).To(Equal(http.StatusOK))
				// Expect(context).To(ContainSubstring("deploy was successful"))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(wd))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.Manifest).To(Equal(fmt.Sprintf("---\napplications:\n- name: deployadactyl\n  memory: 256M\n  disk_quota: 256M\n")))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.ArtifactURL).To(Equal("Local Developer App Deploy" + " " + wd))
			})
		})

		Context("when manifest file cannot be found in the extracted zip", func() {
			It("continues to deploy successfully", func() {
				eventManager.EmitCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				blueGreener.PushCall.Returns.Error = nil

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", zipRequest, context)
				Expect(err).To(BeNil())
				Expect(statusCode).To(Equal(http.StatusOK))
				// Expect(context).To(ContainSubstring("deploy was successful"))
			})
		})

		Context("push fails", func() {
			It("rejects the request with a http.StatusInternalServerError Internal Server Error", func() {
				eventManager.EmitCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				By("making bluegreener return an error")
				blueGreener.PushCall.Returns.Error = errors.New("blue green error")

				By("getting a temporary manifest from the working directory")
				wd, err := os.Getwd()
				Expect(err).To(BeNil())

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, wd, zipRequest, context)
				Expect(err).To(MatchError("blue green error"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(wd))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.Manifest).To(Equal(fmt.Sprintf("---\napplications:\n- name: deployadactyl\n  memory: 256M\n  disk_quota: 256M\n")))
				Expect(blueGreener.PushCall.Received.DeploymentInfo.ArtifactURL).To(Equal("Local Developer App Deploy" + " " + wd))
			})
		})

		Context("deploy event handler fails", func() {
			It("rejects the request with a http.StatusInternalServerError Internal Server Error", func() {
				eventManager.EmitCall.Returns.Error = errors.New("event error")
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				By("getting a temporary manifest from the working directory")
				wd, err := os.Getwd()
				Expect(err).To(BeNil())

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, wd, zipRequest, context)
				Expect(err).To(MatchError("an error occurred in the deploy.start event"))
				Expect(statusCode).To(Equal(http.StatusInternalServerError))
				// Expect(context).To(ContainSubstring("event error"))
			})
		})
	})

	Context("when authentication is required and a username and password are provided", func() {
		It("accepts the request with a http.StatusOK OK", func() {
			eventManager.EmitCall.Returns.Error = nil

			By("setting authenticate to true")
			deployer.Config.Environments[environmentName] = config.Environment{Authenticate: true}

			By("setting basic auth")
			username = "username-" + deployer.Randomizer.StringRunes(10)
			password = "password-" + deployer.Randomizer.StringRunes(10)
			req.SetBasicAuth(username, password)

			err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))

			// Expect(context).To(ContainSubstring("deploy was successful"))
			Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
			// Expect(context).To(ContainSubstring(fmt.Sprintf("Username:     %s", username)))
		})
	})

	Context("when authentication is required", func() {
		Context("a username and password are not provided", func() {
			It("rejects the request with a http.StatusUnauthorized unauthorized", func() {
				By("setting authenticate to true")
				deployer.Config.Environments[environmentName] = config.Environment{Authenticate: true}

				By("not setting basic auth")

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("basic auth header not found"))
				Expect(statusCode).To(Equal(http.StatusUnauthorized))

				Expect(eventManager.EmitCall.TimesCalled).To(Equal(0), eventManagerNotEnoughCalls)
			})
		})

		Context("the username and password are incorrect", func() {
			It("rejects the request with a http.StatusUnauthorized unauthorized", func() {
				eventManager.EmitCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath
				fetcher.FetchCall.Returns.Error = nil
				blueGreener.PushCall.Returns.Error = errors.New("push failed: login failed")

				By("setting authenticate to true")
				deployer.Config.Environments[environmentName] = config.Environment{Authenticate: true}

				By("providing false credentials")
				username = "username-" + deployer.Randomizer.StringRunes(10)
				password = "password-" + deployer.Randomizer.StringRunes(10)
				req.SetBasicAuth(username, password)

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
				Expect(err).To(MatchError("push failed: login failed"))
				Expect(statusCode).To(Equal(http.StatusUnauthorized))

				Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
			})
		})
	})

	Context("when authentication is not required", func() {
		It("uses the config username and password and accepts the request with a http.StatusOK OK", func() {
			eventManager.EmitCall.Returns.Error = nil

			By("setting authenticate to true")
			deployer.Config.Environments[environmentName] = config.Environment{Authenticate: false}

			err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))

			// Expect(context).To(ContainSubstring("deploy was successful"))
			Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
			// Expect(context).To(ContainSubstring(fmt.Sprintf("Username:     %s", username)))
		})
	})

	Context("with no environments", func() {
		It("returns an error", func() {
			errorMessage := "environment not found: " + environmentName

			eventManager.EmitCall.Returns.Error = nil

			emptyConfiguration := config.Config{
				Username:     "",
				Password:     "",
				Environments: nil,
			}

			deployer = Deployer{emptyConfiguration, blueGreener, fetcher, prechecker, eventManager, randomizerMock, log}
			err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
			// Expect(context).To(ContainSubstring(errorMessage))
			Expect(err).To(MatchError(errorMessage))
			Expect(statusCode).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("deployer prechecker fails", func() {
		It("rejects the request with a http.StatusInternalServerError Internal Server Error", func() {
			prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New(deployAborted)

			err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
			Expect(err).To(MatchError("Deploy aborted, one or more CF foundations unavailable"))
			Expect(statusCode).To(Equal(http.StatusInternalServerError))

			Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
		})
	})

	Describe("deployment output", func() {
		It("shows the user deployment info properties", func() {
			eventManager.EmitCall.Returns.Error = nil

			err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, "", jsonRequest, context)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))

			// Expect(context).To(ContainSubstring(artifactURL))
			// Expect(context).To(ContainSubstring(username))
			// Expect(context).To(ContainSubstring(environmentName))
			// Expect(context).To(ContainSubstring(org))
			// Expect(context).To(ContainSubstring(space))
			// Expect(context).To(ContainSubstring(appName))

			Expect(eventManager.EmitCall.TimesCalled).To(Equal(3), eventManagerNotEnoughCalls)
		})
	})
})
