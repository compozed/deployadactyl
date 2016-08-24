// Package bluegreen is responsible for concurrently pushing an application to multiple Cloud Foundry instances.
package bluegreen

import (
	"bytes"
	"fmt"

	"github.com/compozed/deployadactyl/config"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	pushFailedRollbackTriggered     = "push failed: rollback triggered"
	pushFailedNoRollbackFirstDeploy = "push failed: first deploy, rollback not enabled"
	loginFailed                     = "push failed: login failed"
)

// BlueGreen has a PusherCreator to creater pushers for blue green deployments.
type BlueGreen struct {
	PusherCreator I.PusherFactory
	Log           *logging.Logger
}

// Push will login to all the Cloud Foundry instances provided in the Config and then push the application to all the instances concurrently.
// If the application fails to start in any of the instances it handles rolling back the application in every instance, unless this is the first deploy and disable rollback is enabled.
func (bg BlueGreen) Push(environment config.Environment, appPath string, deploymentInfo S.DeploymentInfo, fw I.FlushWriter) error {
	var responseLogs []byte

	actors := make([]actor, len(environment.Foundations))
	buffers := make([]*bytes.Buffer, len(actors))
	firstDeploy := true

	for i, foundationURL := range environment.Foundations {
		pusher, err := bg.PusherCreator.CreatePusher()
		if err != nil {
			return errors.New(err)
		}
		defer pusher.CleanUp()

		actors[i] = newActor(pusher, foundationURL)
		defer close(actors[i].commands)

		buffers[i] = &bytes.Buffer{}
	}

	failed := bg.loginAll(actors, buffers, deploymentInfo)
	if failed {
		for _, buffer := range buffers {
			buffer.WriteTo(fw)
		}
		return errors.New(loginFailed)
	}

	bg.cleanUpAll(actors, deploymentInfo)

	failed, firstDeploy, responseLogs = bg.pushAll(actors, buffers, appPath, environment.Domain, deploymentInfo)

	combinedOutput := &bytes.Buffer{}
	for _, buffer := range buffers {
		buffer.WriteTo(combinedOutput)
	}
	_, err := combinedOutput.WriteTo(fw)
	if err != nil {
		return errors.New(err)
	}

	defer func() {
		fmt.Fprint(fw, string(responseLogs))
	}()

	// Rollback if deploy failed and this is not the first build or DisableFirstDeployRollback is false
	if failed && (!firstDeploy || (firstDeploy && !environment.DisableFirstDeployRollback)) {
		bg.rollbackAll(actors, deploymentInfo, firstDeploy)
	}

	if failed {
		return errors.Errorf(pushFailedNoRollbackFirstDeploy + "\n" + combinedOutput.String())
	}

	bg.finishPushAll(actors, deploymentInfo)

	return nil
}

func (bg BlueGreen) loginAll(actors []actor, buffers []*bytes.Buffer, deploymentInfo S.DeploymentInfo) bool {
	failed := false

	for i, a := range actors {
		buffer := buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Login(foundationURL, deploymentInfo, buffer)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return failed
}

func (bg BlueGreen) cleanUpAll(actors []actor, deploymentInfo S.DeploymentInfo) {
	foundVenerable := false
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			if pusher.Exists(deploymentInfo.AppName + "-venerable") {
				foundVenerable = true
				return pusher.Rollback(deploymentInfo, false)
			}
			return nil
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
	if foundVenerable {
		bg.Log.Error("cleaned up venerable instances of " + deploymentInfo.AppName)
	}
}

func (bg BlueGreen) pushAll(actors []actor, buffers []*bytes.Buffer, appPath, domain string, deploymentInfo S.DeploymentInfo) (bool, bool, []byte) {
	failed, firstDeploy := false, true
	var responseLogs []byte

	for i, a := range actors {
		buffer := buffers[i]
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			if pusher.Exists(deploymentInfo.AppName) {
				firstDeploy = false
			}
			logs, err := pusher.Push(appPath, domain, deploymentInfo, buffer)
			if logs != nil {
				responseLogs = append(responseLogs, []byte(fmt.Sprintf("\nCloud Foundry Logs for %s at %s\n------------------------------------------------------------\n%s\n", deploymentInfo.AppName, foundationURL, logs))...)
			}
			return err
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
			failed = true
		}
	}

	return failed, firstDeploy, responseLogs
}

func (bg BlueGreen) rollbackAll(actors []actor, deploymentInfo S.DeploymentInfo, firstDeploy bool) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.Rollback(deploymentInfo, firstDeploy)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}
func (bg BlueGreen) finishPushAll(actors []actor, deploymentInfo S.DeploymentInfo) {
	for _, a := range actors {
		a.commands <- func(pusher I.Pusher, foundationURL string) error {
			return pusher.FinishPush(deploymentInfo, foundationURL)
		}
	}
	for _, a := range actors {
		if err := <-a.errs; err != nil {
			bg.Log.Error(err.Error())
		}
	}
}
