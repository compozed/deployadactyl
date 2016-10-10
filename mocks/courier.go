package mocks

// Courier handmade mock for tests.
type Courier struct {
	LoginCall struct {
		Received struct {
			FoundationURL string
			Username      string
			Password      string
			Org           string
			Space         string
			SkipSSL       bool
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	DeleteCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	PushCall struct {
		Received struct {
			AppName   string
			AppPath   string
			Instances uint16
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	RenameCall struct {
		Received struct {
			AppName          string
			AppNameVenerable string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	LogsCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	MapRouteCall struct {
		Received struct {
			AppName string
			Domain  string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	ExistsCall struct {
		Received struct {
			AppName string
		}
		Returns struct {
			Bool bool
		}
	}

	CupsCall struct {
		Received struct {
			AppName string
			Body    string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	UupsCall struct {
		Received struct {
			AppName string
			Body string
		}
		Returns struct {
			Output []byte
			Error error
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}
}

// Login mock method.
func (c *Courier) Login(api, username, password, org, space string, skipSSL bool) ([]byte, error) {
	c.LoginCall.Received.FoundationURL = api
	c.LoginCall.Received.Username = username
	c.LoginCall.Received.Password = password
	c.LoginCall.Received.Org = org
	c.LoginCall.Received.Space = space
	c.LoginCall.Received.SkipSSL = skipSSL

	return c.LoginCall.Returns.Output, c.LoginCall.Returns.Error
}

// Delete mock method.
func (c *Courier) Delete(appName string) ([]byte, error) {
	c.DeleteCall.Received.AppName = appName

	return c.DeleteCall.Returns.Output, c.DeleteCall.Returns.Error
}

// Push mock method.
func (c *Courier) Push(appName, appLocation string, instances uint16) ([]byte, error) {
	c.PushCall.Received.AppName = appName
	c.PushCall.Received.AppPath = appLocation
	c.PushCall.Received.Instances = instances

	return c.PushCall.Returns.Output, c.PushCall.Returns.Error
}

// Rename mock method.
func (c *Courier) Rename(appName, newAppName string) ([]byte, error) {
	c.RenameCall.Received.AppName = appName
	c.RenameCall.Received.AppNameVenerable = newAppName

	return c.RenameCall.Returns.Output, c.RenameCall.Returns.Error
}

// MapRoute mock method.
func (c *Courier) MapRoute(appName, domain string) ([]byte, error) {
	c.MapRouteCall.Received.AppName = appName
	c.MapRouteCall.Received.Domain = domain

	return c.MapRouteCall.Returns.Output, c.MapRouteCall.Returns.Error
}

// Logs mock method.
func (c *Courier) Logs(appName string) ([]byte, error) {
	c.MapRouteCall.Received.AppName = appName

	return c.MapRouteCall.Returns.Output, c.MapRouteCall.Returns.Error
}

// Exists mock method.
func (c *Courier) Exists(appName string) bool {
	c.ExistsCall.Received.AppName = appName

	return c.ExistsCall.Returns.Bool
}

// Cups mock method
func (c *Courier) Cups(appName string, body string) ([]byte, error) {
	c.CupsCall.Received.AppName = appName
	c.CupsCall.Received.Body = body

	return c.CupsCall.Returns.Output, c.CupsCall.Returns.Error
}

// Uups mock method
func (c *Courier) Uups(appName string, body string) ([]byte, error) {
	c.UupsCall.Received.AppName = appName
	c.UupsCall.Received.Body = body

	return c.UupsCall.Returns.Output, c.UupsCall.Returns.Error
}

// CleanUp mock method.
func (c *Courier) CleanUp() error {
	return c.CleanUpCall.Returns.Error
}
