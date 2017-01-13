package manifestro

import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/op/go-logging"
)

type manifestYaml struct {
	Applications []Application `yaml:"applications"`
}

type Application struct {
	Name              string `yaml:"name"`
	Memory            string `yaml:"memory,omitempty"`
	Timeout           *uint16 `yaml:"timeout,omitempty"`
	Instances         *uint16 `yaml:"instances,omitempty"`
	Path              string `yaml:"path,omitempty"`
	Java_opts         string `yaml:"JAVA_OPTS,omitempty"`
	Command           string `yaml:"command,omitempty"`
	Buildpack         string `yaml:"buildpack,omitempty"`
	Disk_quota        string `yaml:"disk_quota,omitempty"`
	Domain            string `yaml:"domain,omitempty"`
	Domains           []string `yaml:"domains,omitempty"`
	Stack             string `yaml:"stack,omitempty"`
	Health_check_type string `yaml:"health-check-type,omitempty"`
	Host              string `yaml:"host,omitempty"`
	Hosts             []string `yaml:"hosts,omitempty"`
	No_Hostname       string `yaml:"no-hostname,omitempty"`
	Routes []struct {
		Route string `yaml:"route,omitempty"`
	} `yaml:"routes,omitempty"`
	Services []string `yaml:"services,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
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

	manifest.Log.Debugf("Attempting to add Map of Environment Variable [%s] to Manifest", name)

	result := true

	if !manifest.parsed {
		result, err = manifest.UnMarshal()
	}

	if !result || err != nil {
		return err
	}

	vars := make(map[string]string)

	if manifest.HasApplications() {

		if manifest.Content.Applications[0].Env != nil {
			vars = manifest.Content.Applications[0].Env
		}

		vars[name] = value
		manifest.Content.Applications[0].Env = vars
	}

	return err
}

func (manifest *Manifest) AddEnvironmentVariables(m map[string]string) (result bool, err error) {

	manifest.Log.Debugf("Attempting to add Map of Environment Variables to Manifest")

	result = false

	if m != nil && len(m) > 0 {
		//#Add Environment Variables if any in the request!
		for k, v := range m {
			err = manifest.AddEnvVar(k, v)
			if err != nil {
				return false, err
			}
		}

		result = true
	}

	return result, err
}

func (manifest *Manifest) HasApplications() (bool) {

	var (
		result bool = true
		err    error
	)

	if !manifest.parsed {
		result, err = manifest.UnMarshal()
	}

	if !result || err != nil {
		return false
	}

	if manifest.Content.Applications != nil && len(manifest.Content.Applications) > 0 {
		return true
	}

	return false
}

func (manifest *Manifest) UnMarshal() (result bool, err error) {
	result = false

	if manifest.Yaml != "" {
		manifest.Log.Debugf("UnMarshaling Yaml => %s", manifest.Yaml)
		//err = yaml.Unmarshal([]byte(manifest.Yaml), &manifest.Content)
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

	//resultBytes, err := yaml.Marshal(manifest.Content)

	resultBytes, err := candiedyaml.Marshal(manifest.Content)

	if err != nil {
		manifest.Log.Errorf("Error occurred marshalling Manifest Yaml! Details: %v", err)
		return manifest.Yaml
	}

	content = string(resultBytes)

	return content

}
