package interfaces

import (
	"github.com/compozed/deployadactyl/config"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener interface.
type BlueGreener interface {
	Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, out FlushWriter) error
}
