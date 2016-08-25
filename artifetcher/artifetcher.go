// Package artifetcher downloads the artifact given a URL.
package artifetcher

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	cannotCreateTempFile           = "cannot create temp file"
	cannotGetURL                   = "cannot GET url"
	cannotCreateArtifactoryRequest = "cannot create artifactory request"
	cannotWriteResponseToFile      = "cannot write response to file"
	cannotCreateTempDirectory      = "cannot create temp directory"
	cannotUnzipArtifact            = "cannot unzip artifact"
)

// Artifetcher fetches artifacts within a file system with an Extractor.
type Artifetcher struct {
	FileSystem *afero.Afero
	Extractor  I.Extractor
	Log        *logging.Logger
}

// Fetch downloads an artifact located at URL.
// It then passes it to the extractor with the manifest for unzipping.
//
// Returns a string to the unzipped artifacts path and an error.
func (a *Artifetcher) Fetch(url, manifest string) (string, error) {
	a.Log.Info("fetching artifact")
	a.Log.Debug("artifact URL: %s", url)

	artifactFile, err := a.FileSystem.TempFile("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempFile, err)
	}
	defer artifactFile.Close()
	defer a.FileSystem.Remove(artifactFile.Name())

	var client = &http.Client{
		Timeout: 4 * time.Minute,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			ExpectContinueTimeout: 2 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		a.Log.Errorf("request: %s", spew.Sdump(req))
		return "", errors.Errorf("%s: %s", cannotCreateArtifactoryRequest, err)
	}

	response, err := client.Do(req)
	if err != nil {
		return "", errors.Errorf("%s: %s: %s", cannotGetURL, url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("%s: %s: %s", cannotGetURL, url, response.Status)
	}

	_, err = io.Copy(artifactFile, response.Body)
	if err != nil {
		a.Log.Error("response: %s", spew.Sdump(response))
		return "", errors.Errorf("%s: %s", cannotWriteResponseToFile, err)
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempDirectory, err)
	}

	err = a.Extractor.Unzip(artifactFile.Name(), unzippedPath, manifest)
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", errors.Errorf("%s: %s", cannotUnzipArtifact, err)
	}

	a.Log.Debug("fetched and unzipped to tempdir %s", unzippedPath)
	return unzippedPath, nil
}

// FetchFromZip fetches files from a compressed zip file.
//
// Returns a string to the unzipped application path and an error.
func (a *Artifetcher) FetchFromZip(requestBody []byte) (string, error) {
	zipFile, err := a.FileSystem.TempFile("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempFile, err)
	}
	defer zipFile.Close()
	defer a.FileSystem.Remove(zipFile.Name())

	a.Log.Info("fetching zip file %s", zipFile.Name())

	f := bytes.NewReader(requestBody)
	if _, err = io.Copy(zipFile, f); err != nil {
		return "", errors.Errorf("%s: %s", cannotWriteResponseToFile, err)
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempDirectory, err)
	}

	err = a.Extractor.Unzip(zipFile.Name(), unzippedPath, "")
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", errors.Errorf("%s: %s", cannotUnzipArtifact, err)
	}

	a.Log.Debug("fetched and unzipped to tempdir %s", unzippedPath)
	return unzippedPath, nil
}
