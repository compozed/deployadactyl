// Package deployer will deploy your application.
package deployer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"bytes"

	"github.com/compozed/deployadactyl/config"
	C "github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer/manifestro"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/spf13/afero"
)

const (
	successfulDeploy = `Your deploy was successful! (^_^)b
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment.
It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.

`

	deploymentOutput = `Deployment Parameters:
Artifact URL: %s,
Username:     %s,
Environment:  %s,
Org:          %s,
Space:        %s,
AppName:      %s`
)

// Deployer contains the bluegreener for deployments, environment variables, a fetcher for artifacts, a prechecker and event manager.
type Deployer struct {
	Config       config.Config
	BlueGreener  I.BlueGreener
	Fetcher      I.Fetcher
	Prechecker   I.Prechecker
	EventManager I.EventManager
	Randomizer   I.Randomizer
	ErrorFinder  I.ErrorFinder
	Log          I.Logger
	FileSystem   *afero.Afero
}

// Deploy takes the deployment information, checks the foundations, fetches the artifact and deploys the application.
func (d Deployer) Deploy(req *http.Request, environment, org, space, appName, contentType string, response io.ReadWriter) (statusCode int, err error) {
	var (
		deploymentInfo         = S.DeploymentInfo{}
		environments           = d.Config.Environments
		authenticationRequired = environments[environment].Authenticate
		deployEventData        = S.DeployEventData{}
		manifest               []byte
		appPath                string
		uuid                   = d.Randomizer.StringRunes(10)
	)
	defer func() { d.FileSystem.RemoveAll(appPath) }()
	d.Log.Debugf("Starting deploy of %s with UUID %s", appName, uuid)
	deploymentLogger := logger.DeploymentLogger{d.Log, uuid}

	e, ok := environments[environment]
	if !ok {
		fmt.Fprintln(response, EnvironmentNotFoundError{environment}.Error())
		return http.StatusInternalServerError, EnvironmentNotFoundError{environment}
	}

	deploymentLogger.Debug("prechecking the foundations")
	err = d.Prechecker.AssertAllFoundationsUp(environments[environment])
	if err != nil {
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, err
	}

	deploymentLogger.Debug("checking for basic auth")
	username, password, ok := req.BasicAuth()
	if !ok {
		if authenticationRequired {
			return http.StatusUnauthorized, BasicAuthError{}
		}
		username = d.Config.Username
		password = d.Config.Password
	}

	if isJSON(contentType) {
		deploymentLogger.Debug("deploying from json request")
		deploymentLogger.Debug("building deploymentInfo")
		deploymentInfo, err = getDeploymentInfo(req.Body)
		if err != nil {
			deploymentLogger.Error(err)
			return http.StatusInternalServerError, err
		}

		if deploymentInfo.Manifest != "" {
			manifest, err = base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
			if err != nil {
				deploymentLogger.Error(err)
				return http.StatusBadRequest, ManifestError{err}
			}
		}

		appPath, err = d.Fetcher.Fetch(deploymentInfo.ArtifactURL, string(manifest))
		if err != nil {
			deploymentLogger.Error(err)
			return http.StatusInternalServerError, err
		}

	} else if isZip(contentType) {
		deploymentLogger.Debug("deploying from zip request")
		appPath, err = d.Fetcher.FetchZipFromRequest(req)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		manifest, _ = d.FileSystem.ReadFile(appPath + "/manifest.yml")

		deploymentInfo.ArtifactURL = appPath
	} else {
		return http.StatusBadRequest, InvalidContentTypeError{}
	}

	deploymentInfo.Username = username
	deploymentInfo.Password = password
	deploymentInfo.Environment = environment
	deploymentInfo.Org = org
	deploymentInfo.Space = space
	deploymentInfo.AppName = appName
	deploymentInfo.UUID = uuid
	deploymentInfo.SkipSSL = environments[environment].SkipSSL
	deploymentInfo.Manifest = string(manifest)
	deploymentInfo.Domain = environments[environment].Domain
	deploymentInfo.AppPath = appPath
	deploymentInfo.CustomParams = make(map[string]interface{})
	deploymentInfo.CustomParams = environments[environment].CustomParams

	instances := manifestro.GetInstances(deploymentInfo.Manifest)
	if instances != nil {
		deploymentInfo.Instances = *instances
	} else {
		deploymentInfo.Instances = environments[environment].Instances
	}

	e, found := environments[deploymentInfo.Environment]
	if !found {
		err = d.EventManager.Emit(S.Event{Type: C.DeployErrorEvent, Data: deployEventData})
		if err != nil {
			deploymentLogger.Error(err)
		}

		err = fmt.Errorf("environment not found: %s", deploymentInfo.Environment)
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, err
	}

	deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	deploymentLogger.Info(deploymentMessage)
	fmt.Fprintln(response, deploymentMessage)

	deployEventData = S.DeployEventData{Response: response, DeploymentInfo: &deploymentInfo, RequestBody: req.Body}

	defer emitDeployFinish(d, deployEventData, response, &err, &statusCode, deploymentLogger)
	defer emitDeploySuccess(d, deployEventData, response, &err, &statusCode, deploymentLogger)

	deploymentLogger.Debugf("emitting a %s event", C.DeployStartEvent)
	err = d.EventManager.Emit(S.Event{Type: C.DeployStartEvent, Data: deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, EventError{C.DeployStartEvent, err}
	}

	err = d.BlueGreener.Push(e, appPath, deploymentInfo, response)
	if err != nil {
		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return http.StatusBadRequest, err
		}
		return http.StatusInternalServerError, err
	}

	deploymentLogger.Infof("successfully deployed application %s", deploymentInfo.AppName)
	fmt.Fprintf(response, "\n%s", successfulDeploy)
	return http.StatusOK, err
}

func getDeploymentInfo(reader io.Reader) (S.DeploymentInfo, error) {
	deploymentInfo := S.DeploymentInfo{}
	err := json.NewDecoder(reader).Decode(&deploymentInfo)
	if err != nil {
		return deploymentInfo, err
	}

	getter := geterrors.WrapFunc(func(key string) string {
		if key == "artifact_url" {
			return deploymentInfo.ArtifactURL
		}
		return ""
	})

	getter.Get("artifact_url")

	err = getter.Err("The following properties are missing")
	if err != nil {
		return S.DeploymentInfo{}, err
	}
	return deploymentInfo, nil
}

func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}

func emitDeployFinish(d Deployer, deployEventData S.DeployEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", C.DeployFinishEvent)

	finishErr := d.EventManager.Emit(S.Event{Type: C.DeployFinishEvent, Data: deployEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)

		*err = fmt.Errorf("%s: %s", *err, EventError{C.DeployFinishEvent, finishErr})
		*statusCode = http.StatusInternalServerError
	}
}

func emitDeploySuccess(d Deployer, deployEventData S.DeployEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	deployEvent := S.Event{Type: C.DeploySuccessEvent, Data: deployEventData}
	if *err != nil {
		tempBuffer := bytes.Buffer{}
		tempBuffer.ReadFrom(response)
		fmt.Fprint(response, tempBuffer.String())

		foundErr := d.ErrorFinder.FindError(tempBuffer.String())
		if foundErr != nil {
			*err = foundErr
		}

		deployEvent.Type = C.DeployFailureEvent
		deployEvent.Error = *err
	}

	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", deployEvent.Type))
	eventErr := d.EventManager.Emit(deployEvent)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", deployEvent.Type, eventErr)
		fmt.Fprintln(response, eventErr)
	}
}
