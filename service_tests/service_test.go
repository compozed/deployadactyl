package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/controller/deployer/eventmanager"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/gin"
	"github.com/go-errors/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	ENDPOINT        = "/v1/apps/:environment/:org/:space/:appName"
	CONFIGPATH      = "./test_config.yml"
	ENVIRONMENTNAME = "test"
	TESTCONFIG      = `---
environments:
- name: Test
  domain: test.example.com
  skip_ssl: true
  foundations:
  - api1.example.com
  - api2.example.com
  - api3.example.com
  - api4.example.com
`
)

var _ = Describe("Service", func() {

	var (
		err                 error
		deployadactylServer *httptest.Server
		artifactServer      *httptest.Server
		creator             Creator
		org                 = randomizer.StringRunes(10)
		space               = randomizer.StringRunes(10)
		appName             = randomizer.StringRunes(10)
	)

	BeforeEach(func() {
		os.Setenv("CF_USERNAME", randomizer.StringRunes(10))
		os.Setenv("CF_PASSWORD", randomizer.StringRunes(10))

		Expect(ioutil.WriteFile(CONFIGPATH, []byte(TESTCONFIG), 0644)).To(Succeed())

		creator, err = New("debug", CONFIGPATH)
		Expect(err).ToNot(HaveOccurred())

		deployadactylHandler := creator.CreateControllerHandler()

		deployadactylServer = httptest.NewServer(deployadactylHandler)

		artifactServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "fixtures/artifact-with-manifest.jar")
		}))
	})

	AfterEach(func() {
		Expect(os.Remove(CONFIGPATH)).To(Succeed())
		deployadactylServer.Close()
		artifactServer.Close()
	})

	Context("mocking the courier and the prechecker", func() {
		It("can deploy an application without the internet", func() {
			j, err := json.Marshal(gin.H{
				"artifact_url": artifactServer.URL,
			})
			Expect(err).ToNot(HaveOccurred())
			jsonBuffer := bytes.NewBuffer(j)

			requestURL := fmt.Sprintf("%s/v1/apps/%s/%s/%s/%s", deployadactylServer.URL, ENVIRONMENTNAME, org, space, appName)
			req, err := http.NewRequest("POST", requestURL, jsonBuffer)
			Expect(err).ToNot(HaveOccurred())

			req.Header.Add("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())

			responseBody, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK), string(responseBody))
		})
	})
})

type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       *logging.Logger
	writer       io.Writer
}

func New(level string, configFilename string) (Creator, error) {
	cfg, err := config.Custom(os.Getenv, configFilename)
	if err != nil {
		return Creator{}, err
	}

	l, err := getLevel(level)
	if err != nil {
		return Creator{}, err
	}

	logger := logger.DefaultLogger(GinkgoWriter, l, "creator")

	eventManager := eventmanager.NewEventManager(logger)

	return Creator{
		config:       cfg,
		eventManager: eventManager,
		logger:       logger,
		writer:       GinkgoWriter,
	}, nil
}

func (c Creator) CreateControllerHandler() *gin.Engine {
	d := c.CreateController()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(c.CreateWriter()))
	r.Use(gin.ErrorLogger())

	r.POST(ENDPOINT, d.Deploy)

	return r
}

func (c Creator) CreateController() controller.Controller {
	return controller.Controller{
		Config:       c.CreateConfig(),
		Deployer:     c.CreateDeployer(),
		Log:          c.CreateLogger(),
		EventManager: c.CreateEventManager(),
		Fetcher:      c.createFetcher(),
	}
}

func (c Creator) CreateRandomizer() I.Randomizer {
	return randomizer.Randomizer{}
}

func (c Creator) CreateDeployer() I.Deployer {
	return deployer.Deployer{
		Config:      c.CreateConfig(),
		BlueGreener: c.CreateBlueGreener(),
		Fetcher: &artifetcher.Artifetcher{
			FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
			Extractor: &extractor.Extractor{
				Log:        c.CreateLogger(),
				FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
			},
			Log: c.CreateLogger(),
		},
		Prechecker:   c.CreatePrechecker(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.CreateRandomizer(),
		Log:          c.CreateLogger(),
	}
}

func (c Creator) createFetcher() I.Fetcher {
	return &artifetcher.Artifetcher{
		FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
		Extractor: &extractor.Extractor{
			Log:        c.CreateLogger(),
			FileSystem: &afero.Afero{Fs: afero.NewOsFs()},
		},
		Log: c.CreateLogger(),
	}
}

func (c Creator) CreatePusher() (I.Pusher, error) {
	courier := &mocks.Courier{}

	courier.LoginCall.Returns.Output = []byte("logged in")
	courier.LoginCall.Returns.Error = nil
	courier.DeleteCall.Returns.Output = []byte("deleted app")
	courier.DeleteCall.Returns.Error = nil
	courier.PushCall.Returns.Output = []byte("pushed app")
	courier.PushCall.Returns.Error = nil
	courier.RenameCall.Returns.Output = []byte("renamed app")
	courier.RenameCall.Returns.Error = nil
	courier.MapRouteCall.Returns.Output = []byte("mapped route")
	courier.MapRouteCall.Returns.Error = nil
	courier.CleanUpCall.Returns.Error = nil

	p := pusher.Pusher{
		Courier: courier,
		Log:     c.CreateLogger(),
	}

	return p, nil
}

func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

func (c Creator) CreateLogger() *logging.Logger {
	return c.logger
}

func (c Creator) CreateConfig() config.Config {
	return c.config
}

func (c Creator) CreatePrechecker() I.Prechecker {
	prechecker := &mocks.Prechecker{}

	prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

	return prechecker
}

func (c Creator) CreateWriter() io.Writer {
	return c.writer
}

func (c Creator) CreateBlueGreener() I.BlueGreener {
	return bluegreen.BlueGreen{
		PusherCreator: c,
		Log:           c.CreateLogger(),
	}
}

func getLevel(level string) (logging.Level, error) {
	if level != "" {
		l, err := logging.LogLevel(level)
		if err != nil {
			return 0, errors.Errorf("unable to get log level: %s. error: %s", level, err.Error())
		}
		return l, nil
	}

	return logging.INFO, nil
}
