// Package deployer will deploy your application.
package deployer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/flushwriter"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	basicAuthHeaderNotFound   = "basic auth header not found"
	environmentNotFound       = "environment not found"
	cannotFetchArtifact       = "cannot fetch artifact"
	invalidArtifact           = "invalid artifact"
	invalidPostRequest        = "invalid POST request"
	cannotOpenManifestFile    = "cannot open manifest file"
	cannotFindManifestFile    = "cannot find manifest file in zip"
	cannotPrintToManifestFile = "cannot print to open manifest file"
	successfulDeploy          = `Your deploy was successful! (^_^)d
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment.
It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.`
	deployStartError  = "an error occurred in the deploy.start event"
	deployFinishError = "an error occurred in the deploy.finish event"
	deploymentOutput  = `Deployment Parameters:
	Artifact URL: %s,
	Username:     %s,
	Environment:  %s,
	Org:          %s,
	Space:        %s,
	AppName:      %s`
	jsonRequestContentType = "application/json"
	zipRequestContentType  = "application/zip"
)

// Deployer contains the bluegreener for deployments, environment variables, a fetcher for artifacts, a prechecker and event manager.
type Deployer struct {
	Config       config.Config
	BlueGreener  I.BlueGreener
	Fetcher      I.Fetcher
	Prechecker   I.Prechecker
	EventManager I.EventManager
	Randomizer   I.Randomizer
	Log          *logging.Logger
}

// Deploy takes the deployment information, checks the foundations, fetches the artifact and deploys the application.
func (d Deployer) Deploy(req *http.Request, environmentName, org, space, appName, appPath, contentType string, g *gin.Context) (err error, statusCode int) {
	var (
		deploymentInfo         = S.DeploymentInfo{}
		environments           = d.Config.Environments
		authenticationRequired = environments[environmentName].Authenticate
		deployEventData        = S.DeployEventData{}
		manifest               []byte
		fw                     = flushwriter.New(g.Writer)
	)

	if isJSONRequest(contentType) {
		deploymentInfo, err = getDeploymentInfo(req.Body)
		if err != nil {
			fmt.Fprintln(&fw, err)
			return err, http.StatusInternalServerError
		}
	}

	username, password, ok := req.BasicAuth()
	if !ok {
		if authenticationRequired {
			return errors.New(basicAuthHeaderNotFound), http.StatusUnauthorized
		}
		username = d.Config.Username
		password = d.Config.Password
	}

	deploymentInfo.Username = username
	deploymentInfo.Password = password
	deploymentInfo.Environment = environmentName
	deploymentInfo.Org = org
	deploymentInfo.Space = space
	deploymentInfo.AppName = appName
	deploymentInfo.UUID = d.Randomizer.StringRunes(128)
	deploymentInfo.SkipSSL = environments[environmentName].SkipSSL

	if isZipRequest(contentType) {
		deploymentInfo.ArtifactURL = "Local Developer App Deploy " + appPath
	}

	deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	d.Log.Debug(deploymentMessage)
	fmt.Fprintln(&fw, deploymentMessage)

	deployEventData = S.DeployEventData{
		Writer:         &fw,
		DeploymentInfo: &deploymentInfo,
		RequestBody:    req.Body,
	}

	if isJSONRequest(contentType) && deploymentInfo.Manifest != "" {
		manifest, err = base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
		if err != nil {
			fmt.Fprintln(&fw, err)
			return errors.New(cannotOpenManifestFile), http.StatusBadRequest
		}
	}
	if isZipRequest(contentType) {
		manifest, err = ioutil.ReadFile(appPath + "/manifest.yml")
		if err != nil {
			fmt.Fprintln(&fw, cannotFindManifestFile)
		}
	}
	deploymentInfo.Manifest = string(manifest)

	defer func() (error, int) {
		deployFinishEvent := S.Event{
			Type: "deploy.finish",
			Data: deployEventData,
		}

		eventErr := d.EventManager.Emit(deployFinishEvent)
		if eventErr != nil {
			fmt.Fprintln(&fw, eventErr)
		}

		if err != nil {
			return err, statusCode
		}

		return nil, http.StatusOK
	}()

	deployStartEvent := S.Event{
		Type: "deploy.start",
		Data: deployEventData,
	}

	err = d.EventManager.Emit(deployStartEvent)
	if err != nil {
		fmt.Fprintln(&fw, err)
		return errors.New(deployStartError), http.StatusInternalServerError
	}

	deployEventData = S.DeployEventData{
		Writer:         &fw,
		DeploymentInfo: &deploymentInfo,
	}

	environment, found := environments[deploymentInfo.Environment]
	if !found {
		var deployEvent = S.Event{
			Type: "deploy.error",
			Data: deployEventData,
		}

		err = d.EventManager.Emit(deployEvent)
		if err != nil {
			fmt.Fprintln(&fw, err)
		}

		err = errors.Errorf("%s: %s", environmentNotFound, deploymentInfo.Environment)
		fmt.Fprintln(&fw, err)
		return err, http.StatusInternalServerError
	}

	err = d.Prechecker.AssertAllFoundationsUp(environment)
	if err != nil {
		fmt.Fprintln(&fw, err)
		return errors.New(err), http.StatusInternalServerError
	}

	if isJSONRequest(contentType) {
		appPath, err = d.Fetcher.Fetch(deploymentInfo.ArtifactURL, deploymentInfo.Manifest)
		if err != nil {
			fmt.Fprintln(&fw, err)
			return err, http.StatusInternalServerError
		}
		defer os.RemoveAll(appPath)
	}

	defer func() {
		var deployEvent = S.Event{
			Type: "deploy.success",
			Data: deployEventData,
		}

		if err != nil {
			deployEvent.Type = "deploy.failure"
		}

		eventErr := d.EventManager.Emit(deployEvent)
		if eventErr != nil {
			fmt.Fprintln(&fw, eventErr)
		}
	}()

	err = d.BlueGreener.Push(environment, appPath, deploymentInfo, &fw)
	if err != nil {
		fmt.Fprintln(&fw, err)
		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return err, http.StatusUnauthorized
		}
		return err, http.StatusInternalServerError
	}

	fmt.Fprintln(&fw, fmt.Sprintf("\n%s", successfulDeploy))
	return err, http.StatusOK
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

func isZipRequest(contentType string) bool {
	return contentType == zipRequestContentType
}

func isJSONRequest(contentType string) bool {
	return contentType == jsonRequestContentType
}
