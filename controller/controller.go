// Package controller is responsible for handling requests from the Server.
package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/gin-gonic/gin"
)

// Controller is used to determine the type of request and process it accordingly.
type Controller struct {
	Deployer I.Deployer
	Log      I.Logger
}

// Deploy checks the request content type and passes it to the Deployer.
func (c *Controller) Deploy(g *gin.Context) {
	c.Log.Debugf("Request originated from: %+v", g.Request.RemoteAddr)

	response := &bytes.Buffer{}

	defer io.Copy(g.Writer, response)
	statusCode, err := c.Deployer.Deploy(
		g.Request,
		g.Param("environment"),
		g.Param("org"),
		g.Param("space"),
		g.Param("appName"),
		g.Request.Header.Get("Content-Type"),
		response,
	)
	if err != nil {
		g.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(response, "cannot deploy application: %s\n", err)
		return
	}

	g.Writer.WriteHeader(statusCode)
}
