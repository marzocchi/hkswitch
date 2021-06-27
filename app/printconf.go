package app

import (
	"fmt"
	"mrz.io/hkswitch/app/config"
	"mrz.io/hkswitch/app/systemd"
	"os"
	"path/filepath"
)

func PrintConf(initType, configFile string, envVars []string) error {
	if initType != "systemd" && initType != "launchd" {
		return fmt.Errorf("only launchd and systemd are supported")
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	binPath, err := os.Executable()
	if err != nil {
		return err
	}

	configPath, err := filepath.Abs(configFile)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	conf := &systemd.Unit{
		Description: cfg.Bridge.Name,
		WorkingDir:  workingDir,
		CommandLine: fmt.Sprintf("%q %q", binPath, configPath),
		Command:     []string{binPath, configFile},
		Home:        home,
	}

	for _, envVar := range envVars {
		conf.Env = append(conf.Env, systemd.EnvVar{
			Name:  envVar,
			Value: os.Getenv(envVar),
		})
	}

	if err := conf.Write(os.Stdout, initType); err != nil {
		return err
	}

	return nil
}
