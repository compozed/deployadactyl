package deployer

import "fmt"

type BasicAuthError struct{}

func (e BasicAuthError) Error() string {
	return "basic auth header not found"
}

type ManifestError struct {
	Err error
}

func (e ManifestError) Error() string {
	return fmt.Sprintf("base64 encoded manifest could not be decoded: %s", e.Err)
}

type InvalidContentTypeError struct{}

func (e InvalidContentTypeError) Error() string {
	return "must be application/json or application/zip"
}

type EventError struct {
	Type string
	Err  error
}

func (e EventError) Error() string {
	return fmt.Sprintf("an error occurred in the %s event: %s", e.Type, e.Err)
}

type EnvironmentNotFoundError struct {
	Environment string
}

func (e EnvironmentNotFoundError) Error() string {
	return fmt.Sprintf("environment not found: %s", e.Environment)
}