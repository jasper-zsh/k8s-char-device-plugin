package config

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Config struct {
	Endpoint       string `yaml:"endpoint"`
	ResourcePrefix string `yaml:"resourcePrefix"`
	Devices        []struct {
		Name string `yaml:"name"`
		Path string `yaml:"path"`
	} `yaml:"devices"`
}

func defaultConfig() *Config {
	return &Config{
		Endpoint:       "k8s-char-device-plugin.sock",
		ResourcePrefix: "char-device",
	}
}

func MustLoadConfigFromFile(filePath string) *Config {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logrus.Fatalf("Failed to read config file: %v", err)
	}
	c := defaultConfig()
	err = yaml.Unmarshal(raw, c)
	if err != nil {
		logrus.Fatalf("Failed to unmarshal config file: %v", err)
	}
	return c
}
