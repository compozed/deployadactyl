package courier_test

import (
	"fmt"
	"math/rand"

	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher/courier"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Courier", func() {
	var (
		appName  string
		output   string
		courier  Courier
		executor *mocks.Executor
	)

	BeforeEach(func() {
		appName = "appName-" + randomizer.StringRunes(10)
		output = "output-" + randomizer.StringRunes(10)
		executor = &mocks.Executor{}
		courier = Courier{
			Executor: executor,
		}
	})

	Describe("logging in", func() {
		It("should get a valid Cloud Foundry login command", func() {
			var (
				api          = "api-" + randomizer.StringRunes(10)
				org          = "org-" + randomizer.StringRunes(10)
				password     = "password-" + randomizer.StringRunes(10)
				space        = "space-" + randomizer.StringRunes(10)
				user         = "user-" + randomizer.StringRunes(10)
				skipSSL      = false
				expectedArgs = []string{"login", "-a", api, "-u", user, "-p", password, "-o", org, "-s", space, ""}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Login(api, user, password, org, space, skipSSL)
			Expect(err).ToNot(HaveOccurred())

			Expect(executor.ExecuteCall.Received.Args).To(Equal(expectedArgs))
			Expect(string(out)).To(Equal(output))
		})

		It("can skip ssl validation", func() {
			var (
				api          = "api-" + randomizer.StringRunes(10)
				org          = "org-" + randomizer.StringRunes(10)
				password     = "password-" + randomizer.StringRunes(10)
				space        = "space-" + randomizer.StringRunes(10)
				user         = "user-" + randomizer.StringRunes(10)
				skipSSL      = true
				expectedArgs = []string{"login", "-a", api, "-u", user, "-p", password, "-o", org, "-s", space, "--skip-ssl-validation"}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.Login(api, user, password, org, space, skipSSL)
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
				expectedArgs = []string{"push", appName, "-i", fmt.Sprint(instances)}
			)

			executor.ExecuteInDirectoryCall.Returns.Output = []byte(output)
			executor.ExecuteInDirectoryCall.Returns.Error = nil

			out, err := courier.Push(appName, appLocation, instances)
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
				expectedArgs = []string{"map-route", appName, domain, "-n", appName}
			)

			executor.ExecuteCall.Returns.Output = []byte(output)
			executor.ExecuteCall.Returns.Error = nil

			out, err := courier.MapRoute(appName, domain)
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

	Describe("updating user provided services", func(){
		It("should get a valid Cloud Foundry Uups command", func ()  {
			var (
				hostName = "hostName-" + randomizer.StringRunes(10)
				address = "address-" + randomizer.StringRunes(10)
				body = fmt.Sprintf("{%s:%s}", hostName, address)
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

	Describe("cleaning up executor directories", func() {
		It("should be successful", func() {
			executor.CleanUpCall.Returns.Error = nil

			Expect(courier.CleanUp()).To(Succeed())
		})
	})
})
