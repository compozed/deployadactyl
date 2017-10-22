package mocks

// Courier handmade mock for tests.
type Courier struct {
	TimesCourierCalled int
	LoginCall          struct {
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
			Hostname  string
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

	MapRouteWithPathCall struct {
		TimesCalled int
		Received    struct {
			AppName  []string
			Domain   []string
			Hostname []string
			Path     []string
		}
		Returns struct {
			Output [][]byte
			Error  []error
		}
	}

	MapRouteCall struct {
		TimesCalled int
		Received    struct {
			AppName  []string
			Domain   []string
			Hostname []string
		}
		Returns struct {
			Output [][]byte
			Error  []error
		}
	}

	UnmapRouteCall struct {
		OrderCalled int
		Received    struct {
			AppName  string
			Domain   string
			Hostname string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	DeleteRouteCall struct {
		OrderCalled int
		Received    struct {
			Domain   string
			Hostname string
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
			Body    string
		}
		Returns struct {
			Output []byte
			Error  error
		}
	}

	DomainsCall struct {
		TimesCalled int
		Returns     struct {
			Domains []string
			Error   error
		}
	}

	CleanUpCall struct {
		Returns struct {
			Error error
		}
	}
}

// Login mock method.
func (c *Courier) Login(foundationURL, username, password, org, space string, skipSSL bool) ([]byte, error) {
	c.LoginCall.Received.FoundationURL = foundationURL
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
func (c *Courier) Push(appName, appLocation, hostname string, instances uint16) ([]byte, error) {
	c.PushCall.Received.AppName = appName
	c.PushCall.Received.AppPath = appLocation
	c.PushCall.Received.Hostname = hostname
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
func (c *Courier) MapRouteWithPath(appName, domain, hostname, path string) ([]byte, error) {
	defer func() { c.MapRouteCall.TimesCalled++ }()

	c.MapRouteWithPathCall.Received.AppName = append(c.MapRouteWithPathCall.Received.AppName, appName)
	c.MapRouteWithPathCall.Received.Domain = append(c.MapRouteWithPathCall.Received.Domain, domain)
	c.MapRouteWithPathCall.Received.Hostname = append(c.MapRouteWithPathCall.Received.Hostname, hostname)
	c.MapRouteWithPathCall.Received.Path = append(c.MapRouteWithPathCall.Received.Path, path)

	if len(c.MapRouteWithPathCall.Returns.Output) == 0 && len(c.MapRouteWithPathCall.Returns.Error) == 0 {
		return []byte{}, nil
	} else if len(c.MapRouteWithPathCall.Returns.Output) == 0 {
		return []byte{}, c.MapRouteWithPathCall.Returns.Error[c.MapRouteCall.TimesCalled]
	} else if len(c.MapRouteWithPathCall.Returns.Error) == 0 {
		return c.MapRouteWithPathCall.Returns.Output[c.MapRouteWithPathCall.TimesCalled], nil
	}

	return c.MapRouteWithPathCall.Returns.Output[c.MapRouteWithPathCall.TimesCalled], c.MapRouteWithPathCall.Returns.Error[c.MapRouteWithPathCall.TimesCalled]
}

// MapRoute mock method.
func (c *Courier) MapRoute(appName, domain, hostname string) ([]byte, error) {
	defer func() { c.MapRouteCall.TimesCalled++ }()

	c.MapRouteCall.Received.AppName = append(c.MapRouteCall.Received.AppName, appName)
	c.MapRouteCall.Received.Domain = append(c.MapRouteCall.Received.Domain, domain)
	c.MapRouteCall.Received.Hostname = append(c.MapRouteCall.Received.Hostname, hostname)

	if len(c.MapRouteCall.Returns.Output) == 0 && len(c.MapRouteCall.Returns.Error) == 0 {
		return []byte{}, nil
	} else if len(c.MapRouteCall.Returns.Output) == 0 {
		return []byte{}, c.MapRouteCall.Returns.Error[c.MapRouteCall.TimesCalled]
	} else if len(c.MapRouteCall.Returns.Error) == 0 {
		return c.MapRouteCall.Returns.Output[c.MapRouteCall.TimesCalled], nil
	}

	return c.MapRouteCall.Returns.Output[c.MapRouteCall.TimesCalled], c.MapRouteCall.Returns.Error[c.MapRouteCall.TimesCalled]
}

// UnmapRoute mock method.
func (c *Courier) UnmapRoute(appName, domain, hostname string) ([]byte, error) {
	defer func() { c.TimesCourierCalled++ }()

	c.UnmapRouteCall.OrderCalled = c.TimesCourierCalled
	c.UnmapRouteCall.Received.AppName = appName
	c.UnmapRouteCall.Received.Domain = domain
	c.UnmapRouteCall.Received.Hostname = hostname

	return c.UnmapRouteCall.Returns.Output, c.UnmapRouteCall.Returns.Error
}

// DeleteRoute mock method.
func (c *Courier) DeleteRoute(domain, hostname string) ([]byte, error) {
	defer func() { c.TimesCourierCalled++ }()

	c.DeleteRouteCall.OrderCalled = c.TimesCourierCalled
	c.DeleteRouteCall.Received.Domain = domain
	c.DeleteRouteCall.Received.Hostname = hostname

	return c.DeleteRouteCall.Returns.Output, c.DeleteRouteCall.Returns.Error
}

// Logs mock method.
func (c *Courier) Logs(appName string) ([]byte, error) {
	c.LogsCall.Received.AppName = appName

	return c.LogsCall.Returns.Output, c.LogsCall.Returns.Error
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

// Domains mock method.
func (c *Courier) Domains() ([]string, error) {
	defer func() { c.DomainsCall.TimesCalled++ }()

	return c.DomainsCall.Returns.Domains, c.DomainsCall.Returns.Error
}

// CleanUp mock method.
func (c *Courier) CleanUp() error {
	return c.CleanUpCall.Returns.Error
}
