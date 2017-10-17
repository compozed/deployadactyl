package routemapper

import (
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/spf13/afero"
)

// RouteMapper will map additional routes to an application at
// deploy time if they are specified in the manifest.
type RouteMapper struct {
	Courier    I.Courier
	FileSystem *afero.Afero
	Log        I.Logger
}

type manifest struct {
	Applications []application
}

type application struct {
	Routes []route
}

type route struct {
	Route string
}

// OnEvent is triggered by the EventManager and maps additional
// routes from the manifest. It will check if the route is a domain
// in the foundation.
func (r RouteMapper) OnEvent(event S.Event) error {
	r.Log.Debugf("starting route mapper")

	var (
		tempAppWithUUID = event.Data.(S.PushEventData).TempAppWithUUID
		deploymentInfo  = event.Data.(S.PushEventData).DeploymentInfo

		manifestBytes []byte
		err           error
	)

	r.Courier = event.Data.(S.PushEventData).Courier.(I.Courier)

	if deploymentInfo.Manifest != "" {
		manifestBytes = []byte(deploymentInfo.Manifest)
	} else if deploymentInfo.AppPath != "" {
		manifestBytes, err = r.FileSystem.ReadFile(deploymentInfo.AppPath + "/manifest.yml")
		if err != nil {
			r.Log.Errorf("failed to read manifest file: %s", err.Error())
			return ReadFileError{err}
		}
	} else {
		r.Log.Info("finished mapping routes: no manifest found")
		return nil
	}

	m := &manifest{}

	r.Log.Debugf("looking for routes in the manifest")
	err = candiedyaml.Unmarshal(manifestBytes, m)
	if err != nil {
		r.Log.Errorf("failed to parse manifest: %s", err.Error())
		return err
	}

	if m.Applications == nil || len(m.Applications[0].Routes) == 0 {
		r.Log.Info("finished mapping routes: no routes to map")
		return nil
	}

	r.Log.Infof("found %d routes in the manifest", len(m.Applications[0].Routes))

	domains, _ := r.Courier.Domains()

	r.Log.Debugf("mapping routes to %s", tempAppWithUUID)
	return r.routeMapper(m, tempAppWithUUID, domains, deploymentInfo)
}

func isRouteADomainInTheFoundation(route string, domains []string) bool {
	for _, domain := range domains {
		if route == domain {
			return true
		}
	}

	return false
}

func (r RouteMapper) routeMapper(m *manifest, tempAppWithUUID string, domains []string, deploymentInfo *S.DeploymentInfo) error {
	for _, route := range m.Applications[0].Routes {
		s := strings.SplitN(route.Route, ".", 2)
		p := strings.SplitN(s[1], "/", 2)

		if isRouteADomainInTheFoundation(route.Route, domains) {
			output, err := r.Courier.MapRoute(tempAppWithUUID, route.Route, deploymentInfo.AppName)
			if err != nil {
				r.Log.Errorf("failed to map route: %s: %s", route.Route, string(output))
				return MapRouteError{route.Route, output}
			}
		} else if len(s) >= 2 && isRouteADomainInTheFoundation(s[1], domains) {
			output, err := r.Courier.MapRoute(tempAppWithUUID, s[1], s[0])
			if err != nil {
				r.Log.Errorf("failed to map route: %s: %s", route.Route, string(output))
				return MapRouteError{route.Route, output}
			}
		} else if isRouteADomainInTheFoundation(p[0], domains) {
			output, err := r.Courier.MapRouteWithPath(tempAppWithUUID, p[0], s[0], p[1])
			if err != nil {
				r.Log.Error(MapRouteError{route.Route, output})
				return MapRouteError{route.Route, output}
			}

		} else {
			return InvalidRouteError{route.Route}
		}
		r.Log.Infof("mapped route %s to %s", route.Route, tempAppWithUUID)
	}

	r.Log.Info("route mapping successful: finished mapping routes")
	return nil
}
