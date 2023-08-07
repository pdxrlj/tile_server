package config

import (
	"os"
	"path/filepath"
)

func GetRootPath() (string, error) {
	rootPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return rootPath, nil
}

func GetConfigPath() (string, error) {
	root, err := GetRootPath()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(root, "config")

	return configPath, nil
}

func GetCmdPath() (string, error) {
	root, err := GetRootPath()
	if err != nil {
		return "", err
	}
	cmdPath := filepath.Join(root, "cmd")
	return cmdPath, nil
}
