package manifestro

import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/op/go-logging"
)

type manifestYaml struct {
	Applications []struct {
		Name      string `yaml:"name"`
		Memory    string `yaml:"memory"`
		Timeout   *uint16 `yaml:"timeout"`
		Instances *uint16 `yaml:"instances"`
		Path      string `yaml:"path"`
		Java_opts string `yaml:"JAVA_OPTS"`
		Buildpack string `yaml:"buildpack"`
		Env       map[string]string `yaml:"env"`
	} `yaml:"applications"`
}

//Contains state of a manifest
type Manifest struct {
	Yaml    string
	parsed  bool
	Log     *logging.Logger
	Content manifestYaml
}

// GetInstances reads a Cloud Foundry manifest as a string and returns the number of Instances
// defined in the manifest, if there are any.
//
// Returns a point to a uint16. If Instances are not found or less than 1, it returns nil.
func (manifest *Manifest) GetInstances() *uint16 {
	var (
		result = true
		err    error
	)

	if !manifest.parsed {
		result, err = manifest.UnMarshal()
	}

	if !result || err != nil {
		return nil
	}

	if manifest.Content.Applications == nil || manifest.Content.Applications[0].Instances == nil || *manifest.Content.Applications[0].Instances < 1 {
		return nil
	}

	return manifest.Content.Applications[0].Instances
}

func CreateManifest(content string, logger *logging.Logger) (manifest *Manifest, err error) {
	manifest = &Manifest{Yaml: content, Log: logger }
	manifest.UnMarshal()
	return manifest, err
}

func (manifest *Manifest) AddEnvVar(name string, value string) (err error) {

	result := true

	if !manifest.parsed {
		result, err = manifest.UnMarshal()
	}

	if !result || err != nil {
		return err
	}

	vars := make(map[string]string)

	if manifest.Content.Applications != nil && len(manifest.Content.Applications) > 0 {

		if manifest.Content.Applications[0].Env != nil {
			vars = manifest.Content.Applications[0].Env
		}

		vars[name] = value
		manifest.Content.Applications[0].Env = vars

		//Erase Path if it exists. We are using the contents of a temp file system with exploded contents.
		manifest.Content.Applications[0].Path = ""
	}

	return err
}

func (manifest *Manifest) UnMarshal() (result bool, err error) {
	result = false

	if manifest.Yaml != "" {
		manifest.Log.Debugf("UnMarshaling Yaml => %s", manifest.Yaml)
		err = candiedyaml.Unmarshal([]byte(manifest.Yaml), &manifest.Content)
		if err != nil {
			manifest.Log.Errorf("Error Unmarshalling Manifest! Details: %v", err)
		} else {
			manifest.parsed = true
			result = true

			manifest.Log.Debugf("UnMarshalled Manifest Contents = %+v", manifest.Content)
		}
	} else {
		return result, nil
	}

	return result, err
}

func (manifest *Manifest) Marshal() (content string) {
	manifest.Log.Debugf("Marshaling Manifest Contents = %+v", manifest.Content)

	resultBytes, err := candiedyaml.Marshal(manifest.Content)

	if err != nil {
		manifest.Log.Errorf("Error occurred marshalling Manifest Yaml! Details: %v", err)
		return manifest.Yaml
	}

	content = string(resultBytes)

	return content

}
