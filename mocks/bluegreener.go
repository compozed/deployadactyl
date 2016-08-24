package mocks

import (
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// BlueGreener handmade mock for tests.
type BlueGreener struct {
	PushCall struct {
		Received struct {
			Environment    config.Environment
			AppPath        string
			DeploymentInfo S.DeploymentInfo
			FlushWriter    interfaces.FlushWriter
		}
		Returns struct {
			Error error
		}
	}
}

// Push mock method.
func (b *BlueGreener) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, flushWriter interfaces.FlushWriter) error {
	b.PushCall.Received.Environment = environment
	b.PushCall.Received.AppPath = appPath
	b.PushCall.Received.DeploymentInfo = deploymentInfo
	b.PushCall.Received.FlushWriter = flushWriter

	return b.PushCall.Returns.Error
}
