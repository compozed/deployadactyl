package mocks

import (
	"io"
	"os"

	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	"github.com/compozed/deployadactyl/eventmanager"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// ENDPOINT is used by the handler to define the deployment endpoint.
const ENDPOINT = "/v1/apps/:environment/:org/:space/:appName"

// Handmade Creator mock.
// Uses a mock prechecker to skip verifying the foundations are up and running.
// Uses a mock Courier and Executor to mock pushing an application.
// Uses a mock FileSystem to mock writing to the operating system.
type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       I.Logger
	writer       io.Writer
	fileSystem   *afero.Afero
}

func NewCreator(level string, configFilename string) (Creator, error) {
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
		fileSystem:   &afero.Afero{Fs: afero.NewMemMapFs()},
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
		Deployer: c.CreateDeployer(),
		Log:      c.CreateLogger(),
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
			FileSystem: c.CreateFileSystem(),
			Extractor: &extractor.Extractor{
				Log:        c.CreateLogger(),
				FileSystem: c.CreateFileSystem(),
			},
			Log: c.CreateLogger(),
		},
		Prechecker:   c.CreatePrechecker(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.CreateRandomizer(),
		Log:          c.CreateLogger(),
		FileSystem:   c.CreateFileSystem(),
		ErrorFinder:  c.createErrorFinder(),
	}
}

func (c Creator) createFetcher() I.Fetcher {
	return &artifetcher.Artifetcher{
		FileSystem: c.CreateFileSystem(),
		Extractor: &extractor.Extractor{
			Log:        c.CreateLogger(),
			FileSystem: c.CreateFileSystem(),
		},
		Log: c.CreateLogger(),
	}
}

func (c Creator) CreatePusher(deploymentInfo S.DeploymentInfo, response io.ReadWriter) (I.Pusher, error) {
	courier := &Courier{}

	courier.LoginCall.Returns.Output = []byte("logged in\t")
	courier.DeleteCall.Returns.Output = []byte("deleted app\t")
	courier.PushCall.Returns.Output = []byte("pushed app\t")
	courier.RenameCall.Returns.Output = []byte("renamed app\t")
	courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("mapped route\t"))
	courier.ExistsCall.Returns.Bool = true

	p := &pusher.Pusher{
		Courier:        courier,
		DeploymentInfo: deploymentInfo,
		EventManager:   c.CreateEventManager(),
		Response:       response,
		Log:            c.CreateLogger(),
	}

	return p, nil
}

func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

func (c Creator) CreateLogger() I.Logger {
	return c.logger
}

func (c Creator) CreateConfig() config.Config {
	return c.config
}

func (c Creator) CreatePrechecker() I.Prechecker {
	return &Prechecker{}
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

func (c Creator) CreateFileSystem() *afero.Afero {
	return c.fileSystem
}

func (c Creator) createErrorFinder() I.ErrorFinder {
	return &error_finder.ErrorFinder{}
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
