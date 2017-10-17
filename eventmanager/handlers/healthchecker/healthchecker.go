package healthchecker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	C "github.com/compozed/deployadactyl/constants"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
)

// HealthChecker will check an endpoint for a http.StatusOK
type HealthChecker struct {
	// OldURL is the prepend on the foundationURL to replace in order to build the
	// newly pushed application URL.
	// Eg: "api.run.pivotal"
	OldURL string

	// NewUrl is what replaces OldURL in the OnEvent function.
	// Eg: "cfapps"
	NewURL string

	Client  I.Client
	Courier I.Courier
	Log     I.Logger
}

// OnEvent is used for the EventManager to do health checking during deployments.
// It will create the new application URL by combining the tempAppWithUUID to the
// domain URL.
func (h HealthChecker) OnEvent(event S.Event) error {

	if event.Type != C.PushFinishedEvent {
		return WrongEventTypeError{event.Type}
	}

	var (
		tempAppWithUUID = event.Data.(S.PushEventData).TempAppWithUUID
		foundationURL   = event.Data.(S.PushEventData).FoundationURL
		deploymentInfo  = event.Data.(S.PushEventData).DeploymentInfo
	)

	if deploymentInfo.HealthCheckEndpoint == "" {
		return nil
	}

	h.Courier = event.Data.(S.PushEventData).Courier.(I.Courier)

	h.Log.Debugf("starting health check")

	newFoundationURL := strings.Replace(foundationURL, h.OldURL, h.NewURL, 1)
	domain := regexp.MustCompile(fmt.Sprintf("%s.*", h.NewURL)).FindString(newFoundationURL)

	err := h.mapTemporaryRoute(tempAppWithUUID, domain)
	if err != nil {
		return err
	}

	// unmapTemporaryRoute will be called before deleteTemporaryRoute
	defer h.deleteTemporaryRoute(tempAppWithUUID, domain)
	defer h.unmapTemporaryRoute(tempAppWithUUID, domain)

	newFoundationURL = strings.Replace(newFoundationURL, h.NewURL, fmt.Sprintf("%s.%s", tempAppWithUUID, h.NewURL), 1)

	return h.Check(newFoundationURL, deploymentInfo.HealthCheckEndpoint)
}

// Check takes a url and endpoint. It does an http.Get to get the response
// status and returns an error if it is not http.StatusOK.
func (h HealthChecker) Check(url, endpoint string) error {
	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")

	h.Log.Debugf("checking route %s%s", url, endpoint)

	resp, err := h.Client.Get(fmt.Sprintf("%s/%s", url, trimmedEndpoint))
	if err != nil {
		h.Log.Error(ClientError{err})
		return ClientError{err}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		h.Log.Errorf("health check failed for %s/%s", url, trimmedEndpoint)
		return HealthCheckError{resp.StatusCode, endpoint, body}
	}

	h.Log.Infof("health check successful for %s%s", url, endpoint)
	return nil
}

func (h HealthChecker) mapTemporaryRoute(tempAppWithUUID, domain string) error {
	h.Log.Debugf("mapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.MapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to map temporary route: %s", out)
		return MapRouteError{tempAppWithUUID, domain}
	}
	h.Log.Infof("mapped temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) deleteTemporaryRoute(tempAppWithUUID, domain string) error {
	h.Log.Debugf("deleting temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.DeleteRoute(domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to delete temporary route: %s", out)
		return DeleteRouteError{tempAppWithUUID, domain}
	}

	h.Log.Infof("deleted temporary route %s.%s", tempAppWithUUID, domain)

	return nil
}

func (h HealthChecker) unmapTemporaryRoute(tempAppWithUUID, domain string) {
	h.Log.Debugf("unmapping temporary route %s.%s", tempAppWithUUID, domain)

	out, err := h.Courier.UnmapRoute(tempAppWithUUID, domain, tempAppWithUUID)
	if err != nil {
		h.Log.Errorf("failed to unmap temporary route: %s", out)
	} else {
		h.Log.Infof("unmapped temporary route %s.%s", tempAppWithUUID, domain)
	}

	h.Log.Infof("finished health check")
}
