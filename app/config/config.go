package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Metric   Metrics `yaml:"metrics"`
	Bridge   `yaml:"bridge"`
	Services []Service `yaml:"services"`
}

func (c *Config) ServiceNames() (list []string) {
	for _, svc := range c.Services {
		list = append(list, svc.Name)
	}

	return
}

type Metrics struct {
	Address string `yaml:"address"`
}

type Bridge struct {
	Name         string `yaml:"name"`
	Pin          string `yaml:"pin"`
	Port         string `yaml:"port"`
	StorageDir   string `yaml:"storage-dir"`
	Manufacturer string `yaml:"manufacturer"`
	SerialNumber string `yaml:"serial-number"`
	Model        string `yaml:"model"`
	Firmware     string `yaml:"firmware"`
}

type Service struct {
	Name       string   `yaml:"name"`
	Command    []string `yaml:"command,flow"`
	Autostart  bool     `yaml:"autostart"`
	Workdir    string   `yaml:"work-dir"`
	Env        []string `yaml:"env"`
	StopSignal string   `yaml:"stop-signal"`
}

var DefaultConfig = Config{}

func Load(f string) (Config, error) {
	data, err := ioutil.ReadFile(f)

	if err != nil {
		return Config{}, fmt.Errorf("load file %s: %w", f, err)
	}

	cfg := DefaultConfig

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("load file %s: %w", f, err)
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if len(cfg.Services) == 0 {
		return fmt.Errorf("empty services list")
	}

	for i, svc := range cfg.Services {
		if svc.Name == "" {
			return fmt.Errorf("empty service name at %d", i)
		}

		if len(svc.Command) < 1 {
			return fmt.Errorf("empty command line for service %s", svc.Name)
		}
	}

	return nil
}
