package stop

import (
	"bytes"
	"fmt"
	"github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
	"io"
	"net/http"
)

type StopControllerConstructor func(log I.DeploymentLogger, deployer I.Deployer, eventManager I.EventManager, errorFinder I.ErrorFinder, startManagerFactory I.StartManagerFactory, resolver I.AuthResolver, envResolver I.EnvResolver) I.StopController

func NewStopController(l I.DeploymentLogger, d I.Deployer, em I.EventManager, ef I.ErrorFinder, smf I.StopManagerFactory, resolver I.AuthResolver, envResolver I.EnvResolver) I.StopController {
	return &StopController{
		Deployer:           d,
		EventManager:       em,
		ErrorFinder:        ef,
		StopManagerFactory: smf,
		Log:                l,
		AuthResolver:       resolver,
		EnvResolver:        envResolver,
	}
}

type StopController struct {
	Deployer           I.Deployer
	Log                I.DeploymentLogger
	StopManagerFactory I.StopManagerFactory
	EventManager       I.EventManager
	ErrorFinder        I.ErrorFinder
	AuthResolver       I.AuthResolver
	EnvResolver        I.EnvResolver
}

func (c *StopController) StopDeployment(deployment *I.Deployment, data map[string]interface{}, response *bytes.Buffer) (deployResponse I.DeployResponse) {
	cf := deployment.CFContext
	c.Log.Debugf("Preparing to stop %s with UUID %s", cf.Application, c.Log.UUID)

	if data == nil {
		data = make(map[string]interface{})
	}

	environment, err := c.EnvResolver.Resolve(cf.Environment)
	if err != nil {
		fmt.Fprintln(response, err.Error())
		return I.DeployResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      err,
		}
	}
	auth, err := c.AuthResolver.Resolve(deployment.Authorization, environment, c.Log)
	if err != nil {
		return I.DeployResponse{
			StatusCode: http.StatusUnauthorized,
			Error:      err,
		}
	}

	deploymentInfo := &structs.DeploymentInfo{
		Org:          cf.Organization,
		Space:        cf.Space,
		AppName:      cf.Application,
		Environment:  cf.Environment,
		UUID:         c.Log.UUID,
		Domain:       environment.Domain,
		SkipSSL:      environment.SkipSSL,
		CustomParams: environment.CustomParams,
		Username:     auth.Username,
		Password:     auth.Password,
		Data:         data,
	}

	defer c.emitStopFinish(response, c.Log, cf, &auth, &environment, data, &deployResponse)
	defer c.emitStopSuccessOrFailure(response, c.Log, cf, &auth, &environment, data, &deployResponse)

	err = c.EventManager.EmitEvent(StopStartedEvent{
		CFContext:     cf,
		Data:          data,
		Environment:   environment,
		Authorization: auth,
		Response:      response,
		Log:           c.Log,
	})
	if err != nil {
		c.Log.Error(err)
		err = &bluegreen.InitializationError{err}
		return I.DeployResponse{
			StatusCode:     http.StatusInternalServerError,
			Error:          deployer.EventError{Type: "StopStartedEvent", Err: err},
			DeploymentInfo: deploymentInfo,
		}
	}

	deployEventData := structs.DeployEventData{Response: response, DeploymentInfo: deploymentInfo}

	manager := c.StopManagerFactory.StopManager(c.Log, deployEventData)
	return *c.Deployer.Deploy(deploymentInfo, environment, manager, response)
}

func (c StopController) emitStopFinish(response io.ReadWriter, deploymentLogger I.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent
	event = StopFinishedEvent{
		CFContext:     cfContext,
		Authorization: *auth,
		Environment:   *environment,
		Data:          data,
		Response:      response,
		Log:           deploymentLogger,
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	c.EventManager.EmitEvent(event)
}

func (c StopController) emitStopSuccessOrFailure(response io.ReadWriter, deploymentLogger I.DeploymentLogger, cfContext I.CFContext, auth *I.Authorization, environment *structs.Environment, data map[string]interface{}, deployResponse *I.DeployResponse) {
	var event I.IEvent

	if deployResponse.Error != nil {
		c.printErrors(response, &deployResponse.Error)
		event = StopFailureEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Error:         deployResponse.Error,
			Response:      response,
			Log:           deploymentLogger,
		}

	} else {
		event = StopSuccessEvent{
			CFContext:     cfContext,
			Authorization: *auth,
			Environment:   *environment,
			Data:          data,
			Response:      response,
			Log:           deploymentLogger,
		}
	}
	deploymentLogger.Debugf("emitting a %s event", event.Name())
	eventErr := c.EventManager.EmitEvent(event)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", event.Name(), eventErr)
		fmt.Fprintln(response, eventErr)
	}
}

func (c StopController) printErrors(response io.ReadWriter, err *error) {
	tempBuffer := bytes.Buffer{}
	tempBuffer.ReadFrom(response)
	fmt.Fprint(response, tempBuffer.String())

	errors := c.ErrorFinder.FindErrors(tempBuffer.String())
	if len(errors) > 0 {
		*err = errors[0]
		for _, error := range errors {
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
			fmt.Fprintln(response)
			fmt.Fprintln(response, "The following error was found in the above logs: "+error.Error())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Error: "+error.Details()[0])
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Potential solution: "+error.Solution())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
		}
	}
}