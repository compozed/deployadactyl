package interfaces

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Deployer interface.
type Deployer interface {
	Deploy(req *http.Request, environment, org, space, appName, appPath, contentType string, g *gin.Context) (error, int)
}
