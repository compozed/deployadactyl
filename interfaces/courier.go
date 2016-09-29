package interfaces

// Courier interface.
type Courier interface {
	Login(api, username, password, org, space string, skipSSL bool) ([]byte, error)
	Delete(appName string) ([]byte, error)
	Push(appName, appLocation string, instances uint16) ([]byte, error)
	Rename(oldName, newName string) ([]byte, error)
	MapRoute(appName, domain string) ([]byte, error)
	Logs(appName string) ([]byte, error)
	Exists(appName string) bool
	Cups(appName string, body string) ([]byte, error)
	CleanUp() error
}
