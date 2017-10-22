package routemapper_test

import (
	"errors"
	"fmt"
	"strconv"

	C "github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/eventmanager/handlers/routemapper"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	logging "github.com/op/go-logging"
	"github.com/spf13/afero"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Routemapper", func() {

	var (
		randomAppName          string
		randomTemporaryAppName string
		randomFoundationURL    string
		randomDomain           string
		randomPath             string
		randomUsername         string
		randomPassword         string
		randomOrg              string
		randomSpace            string
		randomHostName         string

		deploymentInfo *S.DeploymentInfo
		event          S.Event

		courier   *mocks.Courier
		af        *afero.Afero
		logBuffer *Buffer

		routemapper RouteMapper
	)

	BeforeEach(func() {
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)
		randomTemporaryAppName = "randomTemporaryAppName-" + randomizer.StringRunes(10)

		s := "random-" + randomizer.StringRunes(10)
		randomFoundationURL = fmt.Sprintf("https://api.cf.%s.com", s)
		randomDomain = fmt.Sprintf("apps.%s.com", s)
		randomPath = "randomPath-" + randomizer.StringRunes(5)

		randomUsername = "randomUsername" + randomizer.StringRunes(10)
		randomPassword = "randomPassword" + randomizer.StringRunes(10)
		randomOrg = "randomOrg" + randomizer.StringRunes(10)
		randomSpace = "randomSpace" + randomizer.StringRunes(10)

		randomHostName = "randomHostName" + randomizer.StringRunes(10)

		deploymentInfo = &S.DeploymentInfo{
			Username: randomUsername,
			Password: randomPassword,
			Org:      randomOrg,
			Space:    randomSpace,
			AppName:  randomAppName,
		}

		courier = &mocks.Courier{}
		af = &afero.Afero{Fs: afero.NewMemMapFs()}

		event = S.Event{
			Type: C.PushFinishedEvent,
			Data: S.PushEventData{
				Courier:         courier,
				TempAppWithUUID: randomTemporaryAppName,
				FoundationURL:   randomFoundationURL,
				DeploymentInfo:  deploymentInfo,
			},
		}

		logBuffer = NewBuffer()

		routemapper = RouteMapper{
			FileSystem: af,
			Log:        logger.DefaultLogger(logBuffer, logging.DEBUG, "routemapper_test"),
		}
	})

	Context("when routes in the manifest include hostnames", func() {

		var routes []string

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
				fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
				fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
			}

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  routes:
  - route: %s
  - route: %s
  - route: %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0", randomDomain + "1", randomDomain + "2"}
		})

		It("returns nil", func() {
			err := routemapper.OnEvent(event)

			Expect(err).ToNot(HaveOccurred())
		})

		It("calls map-route for the number of routes", func() {
			routemapper.OnEvent(event)

			for i := 0; i < len(routes); i++ {
				Expect(courier.MapRouteCall.Received.AppName[i]).To(Equal(randomTemporaryAppName))
				Expect(courier.MapRouteCall.Received.Domain[i]).To(Equal(randomDomain + strconv.Itoa(i)))
				Expect(courier.MapRouteCall.Received.Hostname[i]).To(Equal(randomHostName + strconv.Itoa(i)))
			}
		})

		It("prints information to the logs", func() {
			routemapper.OnEvent(event)

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("looking for routes in the manifest"))
			Eventually(logBuffer).Should(Say(fmt.Sprintf("found %s routes in the manifest", strconv.Itoa(len(routes)))))
			Eventually(logBuffer).Should(Say(fmt.Sprintf("mapping routes to %s", randomTemporaryAppName)))
			Eventually(logBuffer).Should(Say(fmt.Sprintf("mapped route %s to %s", routes[0], randomTemporaryAppName)))
			Eventually(logBuffer).Should(Say(fmt.Sprintf("mapped route %s to %s", routes[1], randomTemporaryAppName)))
			Eventually(logBuffer).Should(Say(fmt.Sprintf("mapped route %s to %s", routes[2], randomTemporaryAppName)))
			Eventually(logBuffer).Should(Say("finished mapping routes"))
		})

		Context("when map route fails", func() {
			It("returns an error", func() {
				courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

				courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
				courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

				err := routemapper.OnEvent(event)

				Expect(err).To(MatchError(MapRouteError{routes[0], []byte("map route output")}))
			})
		})

		It("prints output to the logs", func() {
			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

			courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
			courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

			routemapper.OnEvent(event)

			Expect(logBuffer).To(Say("mapping routes"))
			Expect(logBuffer).To(Say("failed to map route"))
			Expect(logBuffer).To(Say("map route output"))
		})
	})

	Context("when a route in the manifest inclues a path", func() {
		var routes []string

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0.%s0/%s0", randomHostName, randomDomain, randomPath),
				fmt.Sprintf("%s1.%s1/%s1", randomHostName, randomDomain, randomPath),
				fmt.Sprintf("%s2.%s2/%s2", randomHostName, randomDomain, randomPath),
			}

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  routes:
  - route: %s
  - route: %s
  - route: %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0", randomDomain + "1", randomDomain + "2"}
		})

		It("returns nil", func() {
			err := routemapper.OnEvent(event)

			Expect(err).ToNot(HaveOccurred())
		})

		It("calls map-route for the number of routes with a path arguement", func() {
			routemapper.OnEvent(event)

			for i := 0; i < len(routes); i++ {
				Expect(courier.MapRouteWithPathCall.Received.Hostname[i]).To(Equal(randomHostName + strconv.Itoa(i)))
				Expect(courier.MapRouteWithPathCall.Received.Domain[i]).To(Equal(randomDomain + strconv.Itoa(i)))
				Expect(courier.MapRouteWithPathCall.Received.Path[i]).To(Equal(randomPath + strconv.Itoa(i)))
			}
		})
	})

	Context("when routes in the manifest do not include hostnames", func() {
		var routes []string

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0", randomDomain),
				fmt.Sprintf("%s1", randomDomain),
				fmt.Sprintf("%s2", randomDomain),
			}

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  routes:
  - route: %s
  - route: %s
  - route: %s`,
				routes[0],
				routes[1],
				routes[2],
			)

		})

		It("calls map-route for the number of routes", func() {
			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0", randomDomain + "1", randomDomain + "2"}

			routemapper.OnEvent(event)

			for i := 0; i < len(routes); i++ {
				Expect(courier.MapRouteCall.Received.AppName[i]).To(Equal(randomTemporaryAppName))
				Expect(courier.MapRouteCall.Received.Domain[i]).To(Equal(randomDomain + strconv.Itoa(i)))
				Expect(courier.MapRouteCall.Received.Hostname[i]).To(Equal(randomAppName))
			}
		})

		Context("when map route fails", func() {
			It("returns an error", func() {
				courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

				courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
				courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

				err := routemapper.OnEvent(event)

				Expect(err).To(MatchError(MapRouteError{routes[0], []byte("map route output")}))
			})

			It("prints output to the logs", func() {
				courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

				courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
				courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

				routemapper.OnEvent(event)

				Expect(logBuffer).To(Say("mapping routes"))
				Expect(logBuffer).To(Say("failed to map route"))
				Expect(logBuffer).To(Say("map route output"))
			})
		})
	})

	Context("when routes are not provided in the manifest", func() {
		It("returns nil and prints no routes to map", func() {
			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example`)

			err := routemapper.OnEvent(event)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("finished mapping routes"))
			Eventually(logBuffer).Should(Say("no routes to map"))
		})
	})

	Context("when a bad yaml is provided", func() {
		It("returns an unmarshall error", func() {
			routes := []string{
				fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
				fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
				fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
			}

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
  - name: example
    routes:
    - route: %s
    route: %s
    - route %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			err := routemapper.OnEvent(event)

			Expect(err.Error()).To(ContainSubstring("while parsing a block mapping"))
			Expect(err.Error()).To(ContainSubstring("did not find expected key"))
		})

		It("prints an error to the logs", func() {
			routes := []string{
				fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
				fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
				fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
			}

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
  - name: example
    routes:
    - route: %s
    route: %s
    - route %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			routemapper.OnEvent(event)

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("failed to parse manifest"))
			Eventually(logBuffer).Should(Say("did not find expected key"))
		})
	})

	Context("when a manifest is not provided in the request or application folder", func() {
		It("does not return an error", func() {
			err := routemapper.OnEvent(event)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("finished mapping routes: no manifest found"))
		})
	})

	Context("when the domain is not found", func() {
		It("returns an error", func() {
			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  routes:
  - route: test.example.com`,
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			err := routemapper.OnEvent(event)

			Expect(err).To(MatchError(InvalidRouteError{"test.example.com"}))
		})
	})

	Context("When the domain is not found and the route is not formatted correctly", func() {
		It("returns an error", func() {

			deploymentInfo.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  routes:
  - route: example`,
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			err := routemapper.OnEvent(event)

			Expect(err).To(MatchError(InvalidRouteError{"example"}))
		})
	})

	Context("when manifest is bundled with the application", func() {
		It("reads the manifest file", func() {
			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			manifest := []byte(fmt.Sprintf(`---
applications:
- name: example
  routes:
  - route: %s.%s`, randomAppName, randomDomain),
			)

			manifestPath, _ := af.TempDir("", "")
			af.WriteFile(manifestPath+"/manifest.yml", manifest, 0644)
			deploymentInfo.AppPath = manifestPath

			routemapper.OnEvent(event)

			Expect(courier.MapRouteCall.Received.AppName[0]).To(Equal(randomTemporaryAppName))
			Expect(courier.MapRouteCall.Received.Domain[0]).To(Equal(randomDomain))
			Expect(courier.MapRouteCall.Received.Hostname[0]).To(Equal(randomAppName))
		})
	})

	Context("when reading the manifest file fails", func() {
		It("returns an error", func() {
			deploymentInfo.AppPath = "manifest.yml"

			err := routemapper.OnEvent(event)

			Expect(err.Error()).To(ContainSubstring("file does not exist"))
		})

		It("prints errors to the log", func() {
			deploymentInfo.AppPath = "manifest.yml"

			routemapper.OnEvent(event)

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("file does not exist"))
		})
	})

	Context("when yaml is provided that is not a cloud foundry manifest", func() {
		It("returns nil and prints no routes to map", func() {
			deploymentInfo.Manifest = fmt.Sprintf(`---
name: hey`)

			err := routemapper.OnEvent(event)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logBuffer).Should(Say("no routes to map"))
		})
	})
})
