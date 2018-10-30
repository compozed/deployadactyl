package routemapper_test

import (
	"errors"
	"fmt"
	"strconv"

	. "github.com/compozed/deployadactyl/eventmanager/handlers/routemapper"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/spf13/afero"

	I "github.com/compozed/deployadactyl/interfaces"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
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
		randomUUID             string

		courier   *mocks.Courier
		af        *afero.Afero
		logBuffer *Buffer

		routemapper        RouteMapper
		routeMapperRequest RouteMapperRequest
	)

	BeforeEach(func() {
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)
		randomTemporaryAppName = "randomTemporaryAppName-" + randomizer.StringRunes(10)

		s := "random-" + randomizer.StringRunes(10)
		randomFoundationURL = fmt.Sprintf("https://api.cf.%s.com", s)
		randomDomain = fmt.Sprintf("apps.%s.com", s)
		randomPath = "randomPath-" + randomizer.StringRunes(5)
		randomUUID = "randomUUID-" + randomizer.StringRunes(5)

		randomUsername = "randomUsername" + randomizer.StringRunes(10)
		randomPassword = "randomPassword" + randomizer.StringRunes(10)
		randomOrg = "randomOrg" + randomizer.StringRunes(10)
		randomSpace = "randomSpace" + randomizer.StringRunes(10)

		randomHostName = "randomHostName" + randomizer.StringRunes(10)

		courier = &mocks.Courier{}
		af = &afero.Afero{Fs: afero.NewMemMapFs()}

		manifest := fmt.Sprintf(`
---
applications:
- name: example
  custom-routes:
  - route: %s
  - route: %s
  - route: %s

  env:
    CONVEYOR: 23432`,
			fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
			fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
			fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
		)

		logBuffer = NewBuffer()

		routeMapperRequest = RouteMapperRequest{
			Logger:          I.DeploymentLogger{Log: I.DefaultLogger(logBuffer, logging.DEBUG, "routemapper_test"), UUID: randomUUID},
			Courier:         courier,
			Manifest:        manifest,
			AppPath:         randomPath,
			TempAppWithUUID: randomTemporaryAppName,
			Application:     randomAppName,
		}

		routemapper = RouteMapper{
			FileSystem: af,
		}
	})

	Context("when routes in the manifest include hostnames", func() {

		var (
			routes []string
		)

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
				fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
				fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
			}

			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0", randomDomain + "1", randomDomain + "2"}
		})

		It("returns nil", func() {
			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err).ToNot(HaveOccurred())
		})

		It("calls map-route for the number of routes", func() {
			routemapper.CustomRouteMapper(routeMapperRequest)

			for i := 0; i < len(routes); i++ {
				Expect(courier.MapRouteCall.Received.AppName[i]).To(Equal(randomTemporaryAppName))
				Expect(courier.MapRouteCall.Received.Domain[i]).To(Equal(randomDomain + strconv.Itoa(i)))
				Expect(courier.MapRouteCall.Received.Hostname[i]).To(Equal(randomHostName + strconv.Itoa(i)))
			}
		})

		It("prints information to the logs", func() {
			routemapper.CustomRouteMapper(routeMapperRequest)

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

				err := routemapper.CustomRouteMapper(routeMapperRequest)

				Expect(err).To(MatchError(MapRouteError{routes[0], []byte("map route output")}))
			})
		})

		It("prints output to the logs", func() {
			courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

			courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
			courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

			routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(logBuffer).To(Say("mapping routes"))
			Expect(logBuffer).To(Say("failed to map route"))
			Expect(logBuffer).To(Say("map route output"))
		})
	})

	Context("when a route in the manifest inclues a path", func() {
		var (
			routes []string
		)

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0.%s0/%s0", randomHostName, randomDomain, randomPath),
				fmt.Sprintf("%s1.%s1/%s1", randomHostName, randomDomain, randomPath),
				fmt.Sprintf("%s2.%s2/%s2", randomHostName, randomDomain, randomPath),
			}

			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  custom-routes:
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
			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err).ToNot(HaveOccurred())
		})

		It("calls map-route for the number of routes with a path argument", func() {
			routemapper.CustomRouteMapper(routeMapperRequest)

			for i := 0; i < len(routes); i++ {
				Expect(courier.MapRouteWithPathCall.Received.Hostname[i]).To(Equal(randomHostName + strconv.Itoa(i)))
				Expect(courier.MapRouteWithPathCall.Received.Domain[i]).To(Equal(randomDomain + strconv.Itoa(i)))
				Expect(courier.MapRouteWithPathCall.Received.Path[i]).To(Equal(randomPath + strconv.Itoa(i)))
			}
		})
	})

	Context("when routes in the manifest do not include hostnames", func() {
		var (
			routes []string
		)

		BeforeEach(func() {
			routes = []string{
				fmt.Sprintf("%s0", randomDomain),
				fmt.Sprintf("%s1", randomDomain),
				fmt.Sprintf("%s2", randomDomain),
			}

			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  custom-routes:
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

			routemapper.CustomRouteMapper(routeMapperRequest)

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

				err := routemapper.CustomRouteMapper(routeMapperRequest)

				Expect(err).To(MatchError(MapRouteError{routes[0], []byte("map route output")}))
			})

			It("prints output to the logs", func() {
				courier.DomainsCall.Returns.Domains = []string{randomDomain + "0"}

				courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
				courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

				routemapper.CustomRouteMapper(routeMapperRequest)

				Expect(logBuffer).To(Say("mapping routes"))
				Expect(logBuffer).To(Say("failed to map route"))
				Expect(logBuffer).To(Say("map route output"))
			})
		})
	})

	Context("when routes are not provided in the manifest", func() {
		It("returns nil and prints no routes to map", func() {
			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
- name: example`)

			err := routemapper.CustomRouteMapper(routeMapperRequest)
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

			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
  - name: example
    custom-routes:
    - route: %s
    route: %s
    - route %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err.Error()).To(ContainSubstring("while parsing a block mapping"))
			Expect(err.Error()).To(ContainSubstring("did not find expected key"))
		})

		It("prints an error to the logs", func() {
			routes := []string{
				fmt.Sprintf("%s0.%s0", randomHostName, randomDomain),
				fmt.Sprintf("%s1.%s1", randomHostName, randomDomain),
				fmt.Sprintf("%s2.%s2", randomHostName, randomDomain),
			}

			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
  - name: example
    custom-routes:
    - route: %s
    route: %s
    - route %s`,
				routes[0],
				routes[1],
				routes[2],
			)

			routemapper.CustomRouteMapper(routeMapperRequest)

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("failed to parse manifest"))
			Eventually(logBuffer).Should(Say("did not find expected key"))
		})
	})

	Context("when a manifest is not provided in the request or application folder", func() {
		It("does not return an error", func() {
			routeMapperRequest.Manifest = ""
			routeMapperRequest.AppPath = ""

			err := routemapper.CustomRouteMapper(routeMapperRequest)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("finished mapping routes: no manifest found"))
		})
	})

	Context("when the domain is not found", func() {
		It("returns an error", func() {
			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  custom-routes:
  - route: test.example.com`,
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err).To(MatchError(InvalidRouteError{"test.example.com"}))
		})
	})

	Context("When the domain is not found and the route is not formatted correctly", func() {
		It("returns an error", func() {
			routeMapperRequest.Manifest = fmt.Sprintf(`
---
applications:
- name: example
  custom-routes:
  - route: example`,
			)

			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err).To(MatchError(InvalidRouteError{"example"}))
		})
	})

	Context("when manifest is bundled with the application", func() {
		It("reads the manifest file", func() {
			courier.DomainsCall.Returns.Domains = []string{randomDomain}

			manifest := []byte(fmt.Sprintf(`---
applications:
- name: example
  custom-routes:
  - route: %s.%s`, randomAppName, randomDomain),
			)

			appPath, _ := af.TempDir("", "")
			af.WriteFile(appPath+"/manifest.yml", manifest, 0644)

			routeMapperRequest.AppPath = appPath
			routeMapperRequest.Manifest = ""

			routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(courier.MapRouteCall.Received.AppName[0]).To(Equal(randomTemporaryAppName))
			Expect(courier.MapRouteCall.Received.Domain[0]).To(Equal(randomDomain))
			Expect(courier.MapRouteCall.Received.Hostname[0]).To(Equal(randomAppName))
		})
	})

	Context("when reading the manifest file fails", func() {
		It("returns an error", func() {
			routeMapperRequest.AppPath = "manifest.yml"
			routeMapperRequest.Manifest = ""

			err := routemapper.CustomRouteMapper(routeMapperRequest)

			Expect(err.Error()).To(ContainSubstring("file does not exist"))
		})

		It("prints errors to the log", func() {
			routeMapperRequest.AppPath = "manifest.yml"
			routeMapperRequest.Manifest = ""

			routemapper.CustomRouteMapper(routeMapperRequest)

			Eventually(logBuffer).Should(Say("starting route mapper"))
			Eventually(logBuffer).Should(Say("file does not exist"))
		})
	})

	Context("when yaml is provided that is not a cloud foundry manifest", func() {
		It("returns nil and prints no routes to map", func() {

			routeMapperRequest.Manifest = fmt.Sprintf(`---
name: hey`)

			err := routemapper.CustomRouteMapper(routeMapperRequest)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logBuffer).Should(Say("no routes to map"))
		})
	})
})
