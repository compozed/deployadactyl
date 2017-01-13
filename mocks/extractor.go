package mocks

// Extractor handmade mock for tests.
type Extractor struct {
	UnzipCall struct {
		Received struct {
			Source      string
			Destination string
			Manifest    string
		}
		Returns struct {
			Error error
		}
	}

	WriteManifestCall struct {
		Received struct {
			Destination string
			Manifest    string
		}
		Returns struct {
			Error error
		}
	}
}


// Unzip mock method.
func (e *Extractor) Unzip(source, destination, manifest string) error {
	e.UnzipCall.Received.Source = source
	e.UnzipCall.Received.Destination = destination
	e.UnzipCall.Received.Manifest = manifest

	return e.UnzipCall.Returns.Error
}

func (e *Extractor) WriteManifest(destination, manifest string) error {
	e.WriteManifestCall.Received.Destination = destination
	e.WriteManifestCall.Received.Manifest = manifest

	return e.WriteManifestCall.Returns.Error
}
