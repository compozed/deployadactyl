package manifestro_test

import (
	. "github.com/compozed/deployadactyl/controller/deployer/manifestro"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/compozed/deployadactyl/logger"
	"github.com/op/go-logging"
)

var _ = Describe("Manifestro", func() {

	var (
		logBuffer = NewBuffer()
		log       = logger.DefaultLogger(logBuffer, logging.DEBUG, "manifrestro tests")
	)
	Context("when manifest is empty", func() {
		It("returns nil", func() {
			manifest, _ := CreateManifest("", log)

			result := manifest.GetInstances()

			Expect(result).To(BeNil())
		})
	})

	Context("when manifest not valid", func() {
		It("returns nil", func() {
			manifest, _ := CreateManifest("bork", log)

			result := manifest.GetInstances()

			Expect(result).To(BeNil())
		})
	})

	Context("when manifest is not empty", func() {
		Context("when instances is not found", func() {
			It("returns nil", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example`, log)

				result := manifest.GetInstances()

				Expect(result).To(BeNil())
			})
		})

		Context("when instances is found", func() {
			It("returns the number of instances", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example
  instances: 2`, log)

				result := manifest.GetInstances()

				Expect(*result).To(Equal(uint16(2)))
			})
		})

		Context("when instances is zero", func() {
			It("returns nil", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example
  instances: 0`, log)

				result := manifest.GetInstances()

				Expect(result).To(BeNil())
			})
		})

		Context("when instances is not a number", func() {
			It("returns nil", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example
  instances: bork`, log)

				result := manifest.GetInstances()

				Expect(result).To(BeNil())
			})
		})

		Context("when applications is not found", func() {
			It("returns nil", func() {
				manifest, _ := CreateManifest(`---
host: dispatch-dev
domain: auth-platform-sandbox.allstate.com
env:
  DISPATCH_BACKEND_URL: https://dispatch-server-dev.apps.nonprod-mpn.ro11.allstate.com
`, log)
				result := manifest.GetInstances()

				Expect(result).To(BeNil())
			})
		})
	})

	Context("when instances is found", func() {
		Context("when there are multiple applications", func() {
			It("returns the number of instances from the first application", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example
  instances: 2
- name: example2
  instances: 4`, log)

				result := manifest.GetInstances()

				Expect(*result).To(Equal(uint16(2)))
			})
		})
	})

	Context("when manifest is empty", func() {
		It("returns empty Applications Collection", func() {
			manifest, err := CreateManifest("", log)

			Expect(len(manifest.Content.Applications)).To(Equal(0))
			Expect(err).To(BeNil())
		})
	})

	Context("when manifest not valid", func() {
		It("returns empty Applications Collection", func() {
			manifest, err := CreateManifest("bork", log)
			Expect(len(manifest.Content.Applications)).To(Equal(0))
			Expect(err).To(BeNil())
		})
	})

	Context("when manifest is not empty", func() {
		Context("when env", func() {
			It("Adds Env Var", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example`, log)

				Expect(len(manifest.Content.Applications)).To(Equal(1))
				Expect(len(manifest.Content.Applications[0].Env)).To(Equal(0))
				manifest.AddEnvVar("bubba", "gump")
				Expect(manifest.Content.Applications[0].Env["bubba"]).To(Equal("gump"))
			})
		})
	})

	Context("when manifest is not empty", func() {
		Context("when env", func() {
			It("Add Multiple Env Var", func() {
				manifest, _ := CreateManifest(`
applications:
- name: example`, log)

				Expect(len(manifest.Content.Applications)).To(Equal(1))
				Expect(len(manifest.Content.Applications[0].Env)).To(Equal(0))

				envs := map[string]string{
					"bubba": "gump",
					"shrimp":"co",
				}
				manifest.AddEnvironmentVariables(envs)

				Expect(len(manifest.Content.Applications[0].Env)).To(Equal(2))
			})
		})
	})

	Context("when manifest is invalid", func() {
		It("manifest has applications is false", func() {
			manifest, _ := CreateManifest(`bork`, log)

			result := manifest.HasApplications()

			Expect(result).To(Equal(false))
		})
	})

	Context("when manifest is emtpy", func() {
		It("manifest has applications is false", func() {
			manifest, _ := CreateManifest(`
applications:`, log)

			result := manifest.HasApplications()

			Expect(result).To(Equal(false))
		})
	})

	Context("when manifest has an application", func() {
		It("manifest has applications is true", func() {
			manifest, _ := CreateManifest(`
applications:
- name: example`, log)

			result := manifest.HasApplications()

			Expect(result).To(Equal(true))
		})
	})

	Context("when valid manifest", func() {
		It("Unmarshalls correctly", func() {

			content := `
---
applications:
- name: some-application
  memory: 1536M
  timeout: 180
  instances: 2
  path: target/some-application.war
  JAVA_OPTS: -Djava.security.egd=file:///dev/urandom
  buildpack: a_java_buildpack
  env:
    REPLACE_LINE_FEED: true`

			manifest, err := CreateManifest(content, log)

			Expect(err).To(BeNil())

			result := manifest.GetInstances()

			Expect(*result).To(Equal(uint16(2)))
		})
	})

	Context("when I create a manifest", func() {
		Context("And then I Marshall it", func() {
			It("marshalls to valid yaml", func() {

				content := `applications:
- name: some-application
  memory: 1536M
  timeout: 180
  instances: 2
`

				manifest := new(Manifest)
				manifest.Log = log
				application := Application{Name: "some-application"}
				manifest.Content.Applications = append(manifest.Content.Applications, application)

				manifest.Content.Applications[0].Memory = "1536M"

				timeout := uint16(180)
				instances := uint16(2)
				manifest.Content.Applications[0].Timeout = &timeout
				manifest.Content.Applications[0].Instances = &instances

				manifestString := manifest.Marshal()

				Expect(content).To(Equal(manifestString))

			})
		})
	})

})
