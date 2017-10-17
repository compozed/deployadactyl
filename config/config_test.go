package config_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
)

const (
	customConfigPath = "./custom_test_config.yml"
	testConfig       = `---
environments:
- name: Test
  domain: test.example.com
  foundations:
  - api1.example.com
  - api2.example.com
  skip_ssl: true
  instances: 3
  custom_params:
    service_now_table_name: u_change
    service_now_column_names:
      change_reason: u_reason
      implementation_plan: u_my_plan
- name: Prod
  domain: example.com
  foundations:
  - api3.example.com
  - api4.example.com
  skip_ssl: false
  custom_params:
    service_now_table_name: change_request
    service_now_column_names:
      change_reason: reason
      implementation_plan: my_plan
`
	badConfigPath = "./test_bad_config.yml"
	noCustomParamsConfig = `---
environments:
- name: Test
  domain: test.example.com
  foundations:
  - api1.example.com
  - api2.example.com
  skip_ssl: true
  instances: 3
- name: Prod
  domain: example.com
  foundations:
  - api3.example.com
  - api4.example.com
  skip_ssl: false
`
)

var _ = Describe("Config", func() {
	var (
		env        *mocks.Env
		envMap     map[string]Environment
		cfUsername string
		cfPassword string
		testColumns map[interface{}]interface{}
		prodColumns map[interface{}]interface{}
	)

	BeforeEach(func() {
		testCustomParams := make(map[string]interface{})
		prodCustomParams := make(map[string]interface{})

		cfUsername = "cfUsername-" + randomizer.StringRunes(10)
		cfPassword = "cfPassword-" + randomizer.StringRunes(10)
		testColumns = make(map[interface{}]interface{})
		prodColumns = make(map[interface{}]interface{})
		testColumns["change_reason"] = "u_reason"
		testColumns["implementation_plan"] = "u_my_plan"
		prodColumns["change_reason"] = "reason"
		prodColumns["implementation_plan"] = "my_plan"

		testCustomParams["service_now_column_names"] = testColumns
		testCustomParams["service_now_table_name"] = "u_change"

		prodCustomParams["service_now_column_names"] = prodColumns
		prodCustomParams["service_now_table_name"] = "change_request"

		env = &mocks.Env{}
		env.GetCall.Returns.Values = map[string]string{}

		envMap = map[string]Environment{
			"test": {
				Name:        "Test",
				Foundations: []string{"api1.example.com", "api2.example.com"},
				Domain:      "test.example.com",
				SkipSSL:     true,
				Instances:   3,
				CustomParams: testCustomParams,
			},
			"prod": {
				Name:        "Prod",
				Foundations: []string{"api3.example.com", "api4.example.com"},
				Domain:      "example.com",
				SkipSSL:     false,
				Instances:   1,
				CustomParams: prodCustomParams,
			},
		}

		Expect(ioutil.WriteFile(customConfigPath, []byte(testConfig), 0644)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(customConfigPath)).To(Succeed())
		Expect(os.RemoveAll(badConfigPath)).To(Succeed())
	})

	Context("when all environment variables are present", func() {
		It("returns a valid config", func() {
			env.GetCall.Returns.Values["CF_USERNAME"] = cfUsername
			env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword
			env.GetCall.Returns.Values["PORT"] = ""

			config, err := Custom(env.Get, customConfigPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Username).To(Equal(cfUsername))
			Expect(config.Password).To(Equal(cfPassword))
			Expect(config.Environments).To(Equal(envMap))
			Expect(config.Port).To(Equal(8080))
		})
	})

	Context("when PORT is in the environment", func() {
		It("uses the value as the port", func() {
			env.GetCall.Returns.Values["CF_USERNAME"] = cfUsername
			env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword
			env.GetCall.Returns.Values["PORT"] = "42"

			config, err := Custom(env.Get, customConfigPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Port).To(Equal(42))
		})
	})

	Context("when an environment variable is missing", func() {
		It("returns an error", func() {
			env.GetCall.Returns.Values["CF_USERNAME"] = ""
			env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword

			_, err := Custom(env.Get, customConfigPath)

			Expect(err).To(MatchError("missing environment variables: CF_USERNAME"))
		})
	})

	Context("when custom params are empty", func() {
		It("should return a valid config with custom params nil", func() {
			env.GetCall.Returns.Values["CF_USERNAME"] = cfUsername
			env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword
			env.GetCall.Returns.Values["PORT"] = ""

			Expect(ioutil.WriteFile(customConfigPath, []byte(noCustomParamsConfig), 0644)).To(Succeed())

			config, err := Custom(env.Get, customConfigPath)

			Expect(err).ToNot(HaveOccurred())
			var nilMap map[string]interface{}
			Expect(config.Environments["test"].CustomParams).To(Equal(nilMap))
		})
	})

	Context("when a bad config is given", func() {
		It("returns an error when environments key is empty", func() {
			env.GetCall.Returns.Values["CF_USERNAME"] = cfUsername
			env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword
			env.GetCall.Returns.Values["PORT"] = "42"

			testBadConfig := `--- ~`
			Expect(ioutil.WriteFile(badConfigPath, []byte(testBadConfig), 0644)).To(Succeed())

			badConfig, err := Custom(env.Get, badConfigPath)
			Expect(err).To(MatchError(EnvironmentsNotSpecifiedError{}))

			Expect(badConfig.Environments).To(BeEmpty())
		})

		Context("missing required parameters", func() {
			It("returns an error when name is missing", func() {
				testBadConfig := `---
environments:
  - name:
    foundations:
    - api1.example.com
`
				Expect(ioutil.WriteFile(badConfigPath, []byte(testBadConfig), 0644)).To(Succeed())

				badConfig, err := Custom(env.Get, badConfigPath)
				Expect(err).To(MatchError(MissingParameterError{}))

				Expect(badConfig.Environments).To(BeEmpty())
			})

			It("returns an error when foundations is missing", func() {
				testBadConfig := `---
environments:
- name: production
  domain: test.example.com
`
				Expect(ioutil.WriteFile(badConfigPath, []byte(testBadConfig), 0644)).To(Succeed())

				badConfig, err := Custom(env.Get, badConfigPath)
				Expect(err).To(MatchError(MissingParameterError{}))

				Expect(badConfig.Environments).To(BeEmpty())
			})
		})

		Context("when the number of instances is zero", func() {
			It("sets the number of instances to one", func() {
				env.GetCall.Returns.Values["CF_USERNAME"] = cfUsername
				env.GetCall.Returns.Values["CF_PASSWORD"] = cfPassword

				testBadConfig := `---
environments:
- name: production
  foundations:
  - api1.example.com
  - api2.example.com
  domain: example.com
  instances: 0
`

				Expect(ioutil.WriteFile(badConfigPath, []byte(testBadConfig), 0644)).To(Succeed())

				badConfig, err := Custom(env.Get, badConfigPath)

				Expect(badConfig.Environments["production"].Instances).To(Equal(uint16(1)))
				Expect(err).ToNot(HaveOccurred())

			})
		})
	})
})
