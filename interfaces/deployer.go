package interfaces

import "github.com/compozed/gin"

// Deployer interface.
type Deployer interface {
	Deploy(g *gin.Context, environment, org, space, appName, appPath, contentType string) (error, int)
}
