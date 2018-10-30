package courier_test

import (
	"fmt"
	"math/rand"

	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/courier"

	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Courier", func() {
	var (
		appName  string
		hostname string
		output   string
		courier  interfaces.Courier
		executor *mocks.Executor
	)

	BeforeEach(func() {
		appName = "appName-" + randomizer.StringRunes(10)
		hostname = "hostname-" + randomizer.StringRunes(10)
		output = "output-" + randomizer.StringRunes(10)
		executor = &mocks.Executor{}
		courier = Courier{
			Executor: executor,
		}
	})

	Describe("logging in", func() {
		It("should get a valid Cloud Foundry login command", func() {
			var (
				foundationURL = "foundationURL-" + randomizer.StringRunes(10)
				org           = "org-" + randomizer.StringRunes(10)
				password      = "password-" + randomizer.StringRunes(10)
				space         = "space-" + randomizer.StringRunes(10)
				user          = "user-" + randomizer.StringRunes(10)
				skipSSL       = false
				expectedArgs  = []string{"login", "-a", foundationURL, "-u", user, "-p", password, "-o", org, "-s", space, ""}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Login(foundationURL, user, password, org, space, skipSSL)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})

		It("can skip ssl validation", func() {
			var (
				foundationURL = "foundationURL-" + randomizer.StringRunes(10)
				org           = "org-" + randomizer.StringRunes(10)
				password      = "password-" + randomizer.StringRunes(10)
				space         = "space-" + randomizer.StringRunes(10)
				user          = "user-" + randomizer.StringRunes(10)
				skipSSL       = true
				expectedArgs  = []string{"login", "-a", foundationURL, "-u", user, "-p", password, "-o", org, "-s", space, "--skip-ssl-validation"}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Login(foundationURL, user, password, org, space, skipSSL)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("starting an app", func() {
		It("should send a valid Cloud Foundry start command", func() {
			expectedArgs := []string{"start", appName}

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Start(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("stopping an app", func() {
		It("should send a valid Cloud Foundry stop command", func() {
			expectedArgs := []string{"stop", appName}

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Stop(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("deleting an app", func() {
		It("should get a valid Cloud Foundry delete command", func() {
			expectedArgs := []string{"delete", appName, "-f"}

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Delete(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("pushing an application", func() {
		It("should get a valid Cloud Foundry push command", func() {
			var (
				appLocation  = "appLocation-" + randomizer.StringRunes(10)
				instances    = uint16(rand.Uint32())
				expectedArgs = []string{"push", appName, "-i", fmt.Sprint(instances), "-n", hostname}
			)

			executor.ExecuteInDirectoryCall.Returns.Output = []byte(output)
			executor.ExecuteInDirectoryCall.Returns.Error = nil

			out, err := courier.Push(appName, appLocation, hostname, instances)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteInDirectoryCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("renaming an app", func() {
		It("should get a valid Cloud Foundry rename command", func() {
			var (
				newAppName   = "newAppName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"rename", appName, newAppName}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Rename(appName, newAppName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("mapping a route", func() {
		It("should get a valid Cloud Foundry map-route command", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				expectedArgs = []string{"map-route", appName, domain, "-n", hostname}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.MapRoute(appName, domain, hostname)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
		It("should get a valid Cloud Foundry map-route command with a path argument", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				path         = "path-" + randomizer.StringRunes(5)
				expectedArgs = []string{"map-route", appName, domain, "-n", hostname, "--path", path}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.MapRouteWithPath(appName, domain, hostname, path)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("unmapping a route", func() {
		It("should get a valid Cloud Foundry unmap-route command", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				expectedArgs = []string{"unmap-route", appName, domain, "-n", hostname}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.UnmapRoute(appName, domain, hostname)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
		It("should get a valid Cloud Foundry unmap-route command with path", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				path         = "path-" + randomizer.StringRunes(5)
				expectedArgs = []string{"unmap-route", appName, domain, "-n", hostname, "--path", path}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.UnmapRouteWithPath(appName, domain, hostname, path)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("deleting a route", func() {
		It("should delete route with hostname and domain", func() {
			var (
				domain       = "domain-" + randomizer.StringRunes(10)
				expectedArgs = []string{"delete-route", domain, "-n", hostname, "-f"}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.DeleteRoute(domain, hostname)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("getting the logs for an application", func() {
		It("should get the recent Cloud Foundry logs", func() {
			expectedArgs := []string{"logs", appName, "--recent"}

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Logs(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("checking for an existing app", func() {
		It("should get a valid cloud foundry exists command", func() {
			expectedArgs := []string{"app", appName}

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			Expect(courier.Exists(appName)).To(BeTrue())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
		})
	})

	Describe("creating user provided services", func() {
		It("should get a valid Cloud Foundry Cups command", func() {
			var (
				hostName     = "hostName-" + randomizer.StringRunes(10)
				address      = "address-" + randomizer.StringRunes(10)
				body         = fmt.Sprintf("{%s:%s}", hostName, address)
				expectedArgs = []string{"cups", appName, "-p", body}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Cups(appName, body)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("creating a service", func() {
		It("should create the database", func() {
			var (
				service      = "service-" + randomizer.StringRunes(10)
				plan         = "plan-" + randomizer.StringRunes(10)
				name         = "name-" + randomizer.StringRunes(10)
				expectedArgs = []string{"create-service", service, plan, name}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.CreateService(service, plan, name)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("binding a service", func() {
		It("should bind the service to the app", func() {
			var (
				appName      = "appName-" + randomizer.StringRunes(10)
				serviceName  = "serviceName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"bind-service", appName, serviceName}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.BindService(appName, serviceName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("unbinding a service", func() {
		It("should unbind the service from the app", func() {
			var (
				appName      = "appName-" + randomizer.StringRunes(10)
				serviceName  = "dbName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"unbind-service", appName, serviceName}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.UnbindService(appName, serviceName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("deleting a service", func() {
		It("should delete the service", func() {
			var (
				serviceName  = "serviceName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"delete-service", serviceName, "-f"}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.DeleteService(serviceName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("restage an app", func() {
		It("should restage the app with the bound service", func() {
			var (
				appName      = "appName-" + randomizer.StringRunes(10)
				expectedArgs = []string{"restage", appName}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Restage(appName)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("updating user provided services", func() {
		It("should get a valid Cloud Foundry Uups command", func() {
			var (
				hostName     = "hostName-" + randomizer.StringRunes(10)
				address      = "address-" + randomizer.StringRunes(10)
				body         = fmt.Sprintf("{%s:%s}", hostName, address)
				expectedArgs = []string{"uups", appName, "-p", body}
			)
			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Uups(appName, body)
			Expect(err).ToNot(HaveOccurred())
			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})
	})

	Describe("getting the list of domains", func() {
		It("gets a valid domains command", func() {
			expectedArgs := []string{"domains"}

			executor.ExecuteCall.Returns.Output = []byte("getting domains in org\nname status\nexample0.com shared\nexample1.com shared\nexample2.com private")
			executor.ExecuteCall.Returns.Error = nil

			domains, err := courier.Domains()
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(domains[0]).To(Equal("example0.com"))
			Expect(domains[1]).To(Equal("example1.com"))
			Expect(domains[2]).To(Equal("example2.com"))
		})
	})

	Describe("cleaning up executor directories", func() {
		It("should be successful", func() {
			executor.CleanUpCall.Returns.Error = nil

			Expect(courier.CleanUp()).To(Succeed())
		})
	})
})
