// Package deployer will deploy your application.
package deployer

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"crypto/tls"
	"log"
	"os"

	"encoding/base64"
	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
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

type SilentDeployer struct {
}

func (d SilentDeployer) Deploy(deploymentInfo *S.DeploymentInfo, env S.Environment, authorization I.Authorization, body io.Reader, actionCreator I.ActionCreator, environment, org, space, appName string, contentType I.DeploymentType, response io.ReadWriter) *I.DeployResponse {
	url := os.Getenv("SILENT_DEPLOY_URL")
	deployResponse := &I.DeployResponse{}

	request, err := http.NewRequest("POST", fmt.Sprintf(url+"/%s/%s/%s", org, space, appName), body)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer request err: %s", err))
		deployResponse.Error = err
	}
	usernamePassword := base64.StdEncoding.EncodeToString([]byte(authorization.Username + ":" + authorization.Password))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", usernamePassword)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Do(request)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer response err: %s", err))
		deployResponse.StatusCode = resp.StatusCode
		deployResponse.Error = err
	}

	deployResponse.StatusCode = resp.StatusCode
	deployResponse.Error = err
	return deployResponse
}

type Deployer struct {
	Config       config.Config
	BlueGreener  I.BlueGreener
	Prechecker   I.Prechecker
	EventManager I.EventManager
	Randomizer   I.Randomizer
	ErrorFinder  I.ErrorFinder
	Log          I.Logger
}

func (d Deployer) Deploy(deploymentInfo *S.DeploymentInfo, env S.Environment, authorization I.Authorization, body io.Reader, actionCreator I.ActionCreator, environment, org, space, appName string, contentType I.DeploymentType, response io.ReadWriter) *I.DeployResponse {

	deploymentLogger := logger.DeploymentLogger{d.Log, deploymentInfo.UUID}

	deployResponse := &I.DeployResponse{
		DeploymentInfo: deploymentInfo,
	}

	deploymentLogger.Debug("prechecking the foundations")
	err := d.Prechecker.AssertAllFoundationsUp(env)
	if err != nil {
		deploymentLogger.Error(err)
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}

	err = actionCreator.SetUp(env.Instances)
	if err != nil {
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}
	defer func() { actionCreator.CleanUp() }()

	err = actionCreator.OnStart()
	if err != nil {
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}

	err = d.BlueGreener.Execute(actionCreator, env, *deploymentInfo, response)

	enableRollback := env.EnableRollback
	if err != nil {
		if !enableRollback {
			deploymentLogger.Errorf("EnableRollback %t, returning status %d and err %s", enableRollback, http.StatusOK, err)
			deployResponse.StatusCode = http.StatusOK
			deployResponse.Error = err
			return deployResponse
		}

		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			deployResponse.StatusCode = http.StatusBadRequest
			deployResponse.Error = err
			return deployResponse
		}

		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}

	deploymentLogger.Infof("successfully deployed application %s", deploymentInfo.AppName)
	fmt.Fprintf(response, "\n%s", successfulDeploy)

	deployResponse.StatusCode = http.StatusOK
	deployResponse.Error = err
	return deployResponse
}
