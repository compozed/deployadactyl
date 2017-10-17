package interfaces

// Courier interface.
type Courier interface {
	Login(foundationURL, username, password, org, space string, skipSSL bool) ([]byte, error)
	Delete(appName string) ([]byte, error)
	Push(appName, appLocation, hostname string, instances uint16) ([]byte, error)
	Rename(oldName, newName string) ([]byte, error)
	MapRoute(appName, domain, hostname string) ([]byte, error)
	MapRouteWithPath(appName, domain, hostname, path string) ([]byte, error)
	UnmapRoute(appName, domain, hostname string) ([]byte, error)
	DeleteRoute(domain, hostname string) ([]byte, error)
	Logs(appName string) ([]byte, error)
	Exists(appName string) bool
	Cups(appName string, body string) ([]byte, error)
	Uups(appName string, body string) ([]byte, error)
	Domains() ([]string, error)
	CleanUp() error
}
