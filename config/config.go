// Package config holds all specified configuration information aggregated from all possible inputs.
package config

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/compozed/deployadactyl/geterrors"
)

const defaultConfigPath = "./config.yml"

// Config is a representation of a config yaml. It can contain multiple Environments.
type Config struct {
	Username     string
	Password     string
	Environments map[string]Environment
	Port         int
}

// Environment is representation of a single environment configuration.
type Environment struct {
	Name         string
	Domain       string
	Foundations  []string `yaml:",flow"`
	Authenticate bool
	SkipSSL      bool `yaml:"skip_ssl"`
	Instances    uint16
	CustomParams map[string]interface{} `yaml:"custom_params"`
}

type configYaml struct {
	Environments []Environment `yaml:",flow"`
}

type foundationYaml struct {
	Foundations []string
}

// Default returns a new Config struct with information from environment variables and the default config file (./config.yml).
func Default(getenv func(string) string) (Config, error) {
	environments, err := getEnvironmentsFromFile(defaultConfigPath)
	if err != nil {
		return Config{}, err
	}
	return createConfig(getenv, environments)
}

// Custom returns a new Config struct with information from environment variables and a custom config file.
func Custom(getenv func(string) string, configPath string) (Config, error) {
	environments, err := getEnvironmentsFromFile(configPath)
	if err != nil {
		return Config{}, err
	}
	return createConfig(getenv, environments)
}

func createConfig(getenv func(string) string, environments map[string]Environment) (Config, error) {
	getter := geterrors.WrapFunc(getenv)

	username := getter.Get("CF_USERNAME")
	password := getter.Get("CF_PASSWORD")

	if err := getter.Err("missing environment variables"); err != nil {
		return Config{}, err
	}

	port, err := getPortFromEnv(getenv)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Username:     username,
		Password:     password,
		Port:         port,
		Environments: environments,
	}
	return config, nil
}

func getPortFromEnv(getenv func(string) string) (int, error) {
	envPort := getenv("PORT")
	if envPort == "" {
		envPort = "8080"
	}

	cfgPort, err := strconv.Atoi(envPort)
	if err != nil {
		return 0, fmt.Errorf("cannot parse $PORT: %s: %s", envPort, err)
	}

	return cfgPort, nil
}

func getEnvironmentsFromFile(filename string) (map[string]Environment, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	foundationConfig, err := parseYamlFromBody(file)
	if err != nil {
		return nil, err
	}

	if foundationConfig.Environments == nil || len(foundationConfig.Environments) == 0 {
		return nil, EnvironmentsNotSpecifiedError{}
	}

	environments := map[string]Environment{}
	for _, environment := range foundationConfig.Environments {
		if environment.Name == "" || environment.Foundations == nil || len(environment.Foundations) == 0 {
			return nil, MissingParameterError{}
		}

		if environment.Instances < 1 {
			environment.Instances = 1
		}

		environments[strings.ToLower(environment.Name)] = environment
	}

	return environments, nil
}

func parseYamlFromBody(data []byte) (configYaml, error) {
	var foundationConfig configYaml

	err := candiedyaml.Unmarshal(data, &foundationConfig)
	if err != nil {
		return configYaml{}, ParseYamlError{err}
	}

	return foundationConfig, nil
}
