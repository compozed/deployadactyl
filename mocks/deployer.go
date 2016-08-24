package mocks

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Deployer handmade mock for tests.
type Deployer struct {
	DeployCall struct {
		TimesCalled int
		Received    struct {
			Request         *http.Request
			EnvironmentName string
			Org             string
			Space           string
			AppName         string
			AppPath         string
			ContentType     string
			Out             io.Writer
		}
		Write struct {
			Output string
		}
		Returns struct {
			Error      error
			StatusCode int
		}
	}
}

// Deploy mock method.
func (d *Deployer) Deploy(req *http.Request, environmentName, org, space, appName, appPath, contentType string, g *gin.Context) (err error, statusCode int) {
	defer func() { d.DeployCall.TimesCalled++ }()

	d.DeployCall.Received.Request = req
	d.DeployCall.Received.EnvironmentName = environmentName
	d.DeployCall.Received.Org = org
	d.DeployCall.Received.Space = space
	d.DeployCall.Received.AppName = appName
	d.DeployCall.Received.AppPath = appPath
	d.DeployCall.Received.ContentType = contentType
	d.DeployCall.Received.Out = g.Writer

	g.Writer.WriteHeader(d.DeployCall.Returns.StatusCode)

	fmt.Fprint(g.Writer, d.DeployCall.Write.Output)

	return d.DeployCall.Returns.Error, d.DeployCall.Returns.StatusCode
}
