package pusher_test

import (
	"errors"
	"fmt"
	"math/rand"

	C "github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/op/go-logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Pusher", func() {
	var (
		pusher       Pusher
		courier      *mocks.Courier
		eventManager *mocks.EventManager

		randomUsername      string
		randomPassword      string
		randomOrg           string
		randomSpace         string
		randomDomain        string
		randomAppPath       string
		randomAppName       string
		randomInstances     uint16
		randomUUID          string
		randomEndpoint      string
		randomFoundationURL string
		randomPushOpts      map[string]string
		tempAppWithUUID     string
		skipSSL             bool
		deploymentInfo      S.DeploymentInfo
		response            *Buffer
		logBuffer           *Buffer
	)

	BeforeEach(func() {
		courier = &mocks.Courier{}
		eventManager = &mocks.EventManager{}

		randomFoundationURL = "randomFoundationURL-" + randomizer.StringRunes(10)
		randomUsername = "randomUsername-" + randomizer.StringRunes(10)
		randomPassword = "randomPassword-" + randomizer.StringRunes(10)
		randomOrg = "randomOrg-" + randomizer.StringRunes(10)
		randomSpace = "randomSpace-" + randomizer.StringRunes(10)
		randomDomain = "randomDomain-" + randomizer.StringRunes(10)
		randomAppPath = "randomAppPath-" + randomizer.StringRunes(10)
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)
		randomEndpoint = "randomEndpoint-" + randomizer.StringRunes(10)
		randomUUID = randomizer.StringRunes(10)
		randomInstances = uint16(rand.Uint32())
		randomPushOpts = map[string]string{"--random-push-opt": randomizer.StringRunes(10)}

		tempAppWithUUID = randomAppName + TemporaryNameSuffix + randomUUID

		response = NewBuffer()
		logBuffer = NewBuffer()

		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

		deploymentInfo = S.DeploymentInfo{
			Username:            randomUsername,
			Password:            randomPassword,
			Org:                 randomOrg,
			Space:               randomSpace,
			AppName:             randomAppName,
			SkipSSL:             skipSSL,
			Instances:           randomInstances,
			Domain:              randomDomain,
			UUID:                randomUUID,
			HealthCheckEndpoint: randomEndpoint,
			PushOpts:            randomPushOpts,
		}

		pusher = Pusher{
			Courier:        courier,
			DeploymentInfo: deploymentInfo,
			EventManager:   eventManager,
			Response:       response,
			Log:            logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
		}
	})

	Describe("logging in", func() {
		Context("when login succeeds", func() {
			It("gives the correct info to the courier", func() {

				Expect(pusher.Login(randomFoundationURL)).To(Succeed())

				Expect(courier.LoginCall.Received.FoundationURL).To(Equal(randomFoundationURL))
				Expect(courier.LoginCall.Received.Username).To(Equal(randomUsername))
				Expect(courier.LoginCall.Received.Password).To(Equal(randomPassword))
				Expect(courier.LoginCall.Received.Org).To(Equal(randomOrg))
				Expect(courier.LoginCall.Received.Space).To(Equal(randomSpace))
				Expect(courier.LoginCall.Received.SkipSSL).To(Equal(skipSSL))
			})

			It("writes the output of the courier to the response", func() {
				courier.LoginCall.Returns.Output = []byte("login succeeded")

				Expect(pusher.Login(randomFoundationURL)).To(Succeed())

				Eventually(response).Should(Say("login succeeded"))
			})
		})

		Context("when login fails", func() {
			It("returns an error", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := pusher.Login(randomFoundationURL)
				Expect(err).To(MatchError(LoginError{randomFoundationURL, []byte("login output")}))
			})

			It("writes the output of the courier to the response", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := pusher.Login(randomFoundationURL)
				Expect(err).To(HaveOccurred())

				Eventually(response).Should(Say("login output"))
			})

			It("logs an error", func() {
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := pusher.Login(randomFoundationURL)
				Expect(err).To(HaveOccurred())

				Eventually(logBuffer).Should(Say(fmt.Sprintf("could not login to %s", randomFoundationURL)))
			})
		})
	})

	Describe("pushing an app", func() {
		Context("when the push succeeds", func() {
			It("pushes the new app", func() {
				courier.PushCall.Returns.Output = []byte("push succeeded")

				Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())

				Expect(courier.PushCall.Received.AppName).To(Equal(tempAppWithUUID))
				Expect(courier.PushCall.Received.AppPath).To(Equal(randomAppPath))
				Expect(courier.PushCall.Received.Hostname).To(Equal(randomAppName))
				Expect(courier.PushCall.Received.Instances).To(Equal(randomInstances))
				Expect(courier.PushCall.Received.PushOpts).To(Equal(randomPushOpts))

				Eventually(response).Should(Say("push succeeded"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("pushing app %s to %s", tempAppWithUUID, randomDomain)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("tempdir for app %s: %s", tempAppWithUUID, randomAppPath)))
				Eventually(logBuffer).Should(Say("output from Cloud Foundry"))
				Eventually(logBuffer).Should(Say("successfully deployed new build"))
			})
		})

		Context("when the push fails", func() {
			It("returns an error", func() {
				courier.PushCall.Returns.Error = errors.New("push error")

				err := pusher.Push(randomAppPath, randomFoundationURL)

				Expect(err).To(MatchError(PushError{}))
			})

			It("gets logs from the courier", func() {
				courier.PushCall.Returns.Output = []byte("push output")
				courier.PushCall.Returns.Error = errors.New("push error")
				courier.LogsCall.Returns.Output = []byte("cf logs")

				Expect(pusher.Push(randomAppPath, randomFoundationURL)).ToNot(Succeed())

				Eventually(response).Should(Say("push output"))
				Eventually(response).Should(Say("cf logs"))

				Eventually(logBuffer).Should(Say("logs from"))
			})

			Context("when the courier log call fails", func() {
				It("returns an error", func() {
					pushErr := errors.New("push error")
					logsErr := errors.New("logs error")

					courier.PushCall.Returns.Error = pushErr
					courier.LogsCall.Returns.Error = logsErr

					err := pusher.Push(randomAppPath, randomFoundationURL)

					Expect(err).To(MatchError(CloudFoundryGetLogsError{pushErr, logsErr}))
				})
			})
		})

		Describe("mapping the load balanced route to the temporary application", func() {
			Context("when a domain is provided", func() {
				It("maps the route to the app", func() {
					Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())

					Expect(courier.MapRouteCall.Received.AppName[0]).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
					Expect(courier.MapRouteCall.Received.Domain[0]).To(Equal(randomDomain))

					Eventually(response).Should(Say(fmt.Sprintf("application route created: %s.%s", randomAppName, randomDomain)))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("mapping route for %s to %s", randomAppName, randomDomain)))
				})
			})

			Context("when a randomDomain is not provided", func() {
				It("does not map the randomDomain", func() {
					courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("mapped route"))
					deploymentInfo.Domain = ""

					pusher = Pusher{
						Courier:        courier,
						DeploymentInfo: deploymentInfo,
						EventManager:   eventManager,
						Response:       response,
						Log:            logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
					}

					Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())

					Expect(courier.MapRouteCall.Received.AppName).To(BeEmpty())
					Expect(courier.MapRouteCall.Received.Domain).To(BeEmpty())

					Eventually(response).ShouldNot(Say("mapped route"))

					Eventually(logBuffer).ShouldNot(Say(fmt.Sprintf("mapping route for %s to", randomAppName)))
				})
			})

			Context("when MapRoute fails", func() {
				It("returns an error", func() {
					courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("unable to map route"))
					courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

					err := pusher.Push(randomAppPath, randomFoundationURL)
					Expect(err).To(MatchError(MapRouteError{[]byte("unable to map route")}))

					Expect(courier.MapRouteCall.Received.AppName[0]).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
					Expect(courier.MapRouteCall.Received.Domain[0]).To(Equal(randomDomain))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("mapping route for %s to %s", randomAppName, randomDomain)))
				})
			})
		})
	})

	Describe("finishing a push", func() {
		It("renames the newly pushed app to the original name", func() {
			Expect(pusher.FinishPush()).To(Succeed())

			Expect(courier.RenameCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
			Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(randomAppName))

			Eventually(logBuffer).Should(Say("renamed %s to %s", tempAppWithUUID, randomAppName))
		})

		Context("when rename fails", func() {
			It("returns an error", func() {
				courier.RenameCall.Returns.Output = []byte("rename output")
				courier.RenameCall.Returns.Error = errors.New("rename error")

				err := pusher.FinishPush()
				Expect(err).To(MatchError(RenameError{randomAppName + TemporaryNameSuffix + randomUUID, []byte("rename output")}))

				Expect(courier.RenameCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
				Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(randomAppName))

				Eventually(logBuffer).Should(Say("could not rename %s to %s", tempAppWithUUID, randomAppName))
			})
		})

		Context("when the app exists", func() {
			BeforeEach(func() {
				courier.ExistsCall.Returns.Bool = true
			})

			It("checks the application exists", func() {
				Expect(pusher.FinishPush()).To(Succeed())

				Expect(courier.ExistsCall.Received.AppName).To(Equal(randomAppName))
			})

			It("unmaps the load balanced route", func() {
				Expect(pusher.FinishPush()).To(Succeed())

				Expect(courier.UnmapRouteCall.Received.AppName).To(Equal(randomAppName))
				Expect(courier.UnmapRouteCall.Received.Domain).To(Equal(randomDomain))
				Expect(courier.UnmapRouteCall.Received.Hostname).To(Equal(randomAppName))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("unmapped route %s", randomAppName)))
			})

			It("deletes the original application ", func() {
				Expect(pusher.FinishPush()).To(Succeed())

				Expect(courier.DeleteCall.Received.AppName).To(Equal(randomAppName))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("deleted %s", randomAppName)))
			})

			Context("when domain is not provided", func() {
				It("does not call unmap route", func() {
					deploymentInfo.Domain = ""

					pusher = Pusher{
						Courier:        courier,
						DeploymentInfo: deploymentInfo,
						EventManager:   eventManager,
						Response:       response,
						Log:            logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
					}

					pusher.FinishPush()

					Expect(courier.UnmapRouteCall.Received.AppName).To(BeEmpty())
					Expect(courier.UnmapRouteCall.Received.Domain).To(BeEmpty())
					Expect(courier.UnmapRouteCall.Received.Hostname).To(BeEmpty())
				})
			})

			Context("when unmapping the route fails", func() {
				It("only logs an error", func() {
					courier.UnmapRouteCall.Returns.Output = []byte("unmap output")
					courier.UnmapRouteCall.Returns.Error = errors.New("Unmap Error")

					err := pusher.FinishPush()
					Expect(err).To(MatchError(UnmapRouteError{randomAppName, []byte("unmap output")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not unmap %s", randomAppName)))
				})
			})

			Context("when deleting the original app fails", func() {
				It("returns an error", func() {
					courier.ExistsCall.Returns.Bool = true
					courier.DeleteCall.Returns.Output = []byte("delete output")
					courier.DeleteCall.Returns.Error = errors.New("delete error")

					err := pusher.FinishPush()
					Expect(err).To(MatchError(DeleteApplicationError{randomAppName, []byte("delete output")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not delete %s", randomAppName)))
				})
			})
		})

		Context("when the application does not exist", func() {
			It("does not delete the non-existant original application", func() {
				courier.ExistsCall.Returns.Bool = false

				err := pusher.FinishPush()
				Expect(err).ToNot(HaveOccurred())

				Expect(courier.DeleteCall.Received.AppName).To(BeEmpty())

				Eventually(logBuffer).ShouldNot(Say("delete"))
			})
		})
	})

	Describe("undoing a push", func() {
		Context("when the app exists", func() {
			BeforeEach(func() {
				courier.ExistsCall.Returns.Bool = true
			})

			It("check that the app exists", func() {
				Expect(pusher.UndoPush()).To(Succeed())
				Expect(courier.ExistsCall.Received.AppName).To(Equal(randomAppName))
			})

			It("deletes the app that was pushed", func() {
				Expect(pusher.UndoPush()).To(Succeed())

				Expect(courier.DeleteCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("rolling back deploy of %s", randomAppName)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("deleted %s", randomAppName)))
			})

			Context("when deleting fails", func() {
				It("returns an error and writes a message to the info log", func() {
					courier.DeleteCall.Returns.Output = []byte("delete call output")
					courier.DeleteCall.Returns.Error = errors.New("delete error")

					err := pusher.UndoPush()
					Expect(err).To(MatchError(DeleteApplicationError{tempAppWithUUID, []byte("delete call output")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not delete %s", tempAppWithUUID)))
				})
			})
		})

		Context("when the app does not exist", func() {
			It("renames the newly built app to the intended application name", func() {
				Expect(pusher.UndoPush()).To(Succeed())

				Expect(courier.RenameCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
				Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(randomAppName))

				Eventually(logBuffer).Should(Say("renamed %s to %s", tempAppWithUUID, randomAppName))
			})

			Context("when renaming fails", func() {
				It("returns an error and writes a message to the info log", func() {
					courier.RenameCall.Returns.Error = errors.New("rename error")
					courier.RenameCall.Returns.Output = []byte("rename error")

					err := pusher.UndoPush()
					Expect(err).To(MatchError(RenameError{tempAppWithUUID, []byte("rename error")}))

					Eventually(logBuffer).Should(Say(fmt.Sprintf("could not rename %s to %s", tempAppWithUUID, randomAppName)))
				})
			})
		})
	})

	Describe("cleaning up temporary directories", func() {
		It("is successful", func() {
			courier.CleanUpCall.Returns.Error = nil

			Expect(pusher.CleanUp()).To(Succeed())
		})
	})

	Describe("event handling", func() {
		Context("when a PushFinishedEvent is emitted", func() {
			It("does not return an error", func() {
				Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())

				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.PushFinishedEvent))
			})

			It("has the temporary app name on the event", func() {
				Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())

				Expect(eventManager.EmitCall.Received.Events[0].Data.(S.PushEventData).TempAppWithUUID).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
			})
		})

		Context("when an event fails", func() {
			It("returns an error", func() {
				eventManager.EmitCall.Returns.Error[0] = errors.New("event manager error")

				err := pusher.Push(randomAppPath, randomFoundationURL)
				Expect(err).To(MatchError("event manager error"))

				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.PushFinishedEvent))
			})
		})
	})
})
