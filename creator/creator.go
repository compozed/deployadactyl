// Package creator creates dependencies upon initialization.
package creator

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/compozed/deployadactyl/artifetcher"
	"github.com/compozed/deployadactyl/artifetcher/extractor"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/actioncreator"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher/courier"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher/courier/executor"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	"github.com/compozed/deployadactyl/controller/deployer/prechecker"
	"github.com/compozed/deployadactyl/eventmanager"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

// ENDPOINT is used by the handler to define the deployment endpoint.
const ENDPOINT = "/v2/deploy/:environment/:org/:space/:appName"

// Creator has a config, eventManager, logger and writer for creating dependencies.
type Creator struct {
	config       config.Config
	eventManager I.EventManager
	logger       I.Logger
	writer       io.Writer
	fileSystem   *afero.Afero
}

// Default returns a default Creator and an Error.
func Default() (Creator, error) {
	cfg, err := config.Default(os.Getenv)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(logging.DEBUG, cfg)
}

// Custom returns a custom Creator with an Error.
func Custom(level string, configFilename string) (Creator, error) {
	l, err := getLevel(level)
	if err != nil {
		return Creator{}, err
	}

	cfg, err := config.Custom(os.Getenv, configFilename)
	if err != nil {
		return Creator{}, err
	}
	return createCreator(l, cfg)
}

// CreateControllerHandler returns a gin.Engine that implements http.Handler.
// Sets up the controller endpoint.
func (c Creator) CreateControllerHandler(controller I.Controller) *gin.Engine {

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(c.createWriter()))
	r.Use(gin.ErrorLogger())

	r.POST(ENDPOINT, controller.RunDeploymentViaHttp)

	return r
}

// CreateListener creates a listener TCP and listens for all incoming requests.
func (c Creator) CreateListener() net.Listener {
	ls, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: c.config.Port,
		Zone: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	return ls
}

// CreateCourier returns a courier with an executor.
func (c Creator) CreateCourier() (I.Courier, error) {
	ex, err := executor.New(c.CreateFileSystem())
	if err != nil {
		return nil, err
	}

	return courier.Courier{
		Executor: ex,
	}, nil
}

// CreateLogger returns a Logger.
func (c Creator) CreateLogger() I.Logger {
	return c.logger
}

// CreateConfig returns a Config.
func (c Creator) CreateConfig() config.Config {
	return c.config
}

// CreateEventManager returns an EventManager.
func (c Creator) CreateEventManager() I.EventManager {
	return c.eventManager
}

// CreateFileSystem returns a file system.
func (c Creator) CreateFileSystem() *afero.Afero {
	return c.fileSystem
}

// CreateHTTPClient return an http client.
func (c Creator) CreateHTTPClient() *http.Client {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return insecureClient
}

func (c Creator) CreateController() I.Controller {
	return &controller.Controller{
		Deployer:             c.createDeployer(),
		SilentDeployer:       c.createSilentDeployer(),
		Log:                  c.CreateLogger(),
		PusherCreatorFactory: c,
	}
}

func (c Creator) createDeployer() I.Deployer {
	return deployer.Deployer{
		Config:       c.CreateConfig(),
		BlueGreener:  c.createBlueGreener(),
		Prechecker:   c.createPrechecker(),
		EventManager: c.CreateEventManager(),
		Randomizer:   c.createRandomizer(),
		ErrorFinder:  c.createErrorFinder(),
		Log:          c.CreateLogger(),
		FileSystem:   c.CreateFileSystem(),
	}
}

func (c Creator) PusherCreator(body io.Reader) I.ActionCreator {
	return actioncreator.PusherCreator{
		CourierCreator: c,
		EventManager:   c.CreateEventManager(),
		Logger:         c.CreateLogger(),
		Fetcher:        c.createFetcher(),
	}
}

func (c Creator) StopperCreator() I.ActionCreator {
	return actioncreator.StopperCreator{
		CourierCreator: c,
		EventManager:   c.CreateEventManager(),
		Logger:         c.CreateLogger(),
	}
}

func (c Creator) createSilentDeployer() I.Deployer {
	return deployer.SilentDeployer{}
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

func (c Creator) createRandomizer() I.Randomizer {
	return randomizer.Randomizer{}
}

func (c Creator) createPrechecker() I.Prechecker {
	return prechecker.Prechecker{
		EventManager: c.CreateEventManager(),
	}
}

func (c Creator) createWriter() io.Writer {
	return c.writer
}

func (c Creator) createBlueGreener() I.BlueGreener {
	return bluegreen.BlueGreen{
		Log: c.CreateLogger(),
	}
}

func (c Creator) createErrorFinder() I.ErrorFinder {
	return &error_finder.ErrorFinder{
		Matchers: c.config.ErrorMatchers,
	}
}

func createCreator(l logging.Level, cfg config.Config) (Creator, error) {
	err := ensureCLI()
	if err != nil {
		return Creator{}, err
	}

	logger := logger.DefaultLogger(os.Stdout, l, "controller")
	eventManager := eventmanager.NewEventManager(logger)

	return Creator{
		cfg,
		eventManager,
		logger,
		os.Stdout,
		&afero.Afero{Fs: afero.NewOsFs()},
	}, nil

}

func ensureCLI() error {
	_, err := exec.LookPath("cf")
	return err
}

func getLevel(level string) (logging.Level, error) {
	if level != "" {
		l, err := logging.LogLevel(level)
		if err != nil {
			return 0, fmt.Errorf("unable to get log level: %s. error: %s", level, err.Error())
		}
		return l, nil
	}

	return logging.INFO, nil
}
