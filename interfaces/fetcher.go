package interfaces

import "net/http"

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchZipFromRequest(*http.Request) (string, error)
	WriteManifest(path, manifest string) (error)
}
