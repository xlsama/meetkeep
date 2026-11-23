package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/config"
)

// launcher is the meetkeep command on the user's path, a script that runs
// the command line kept among the app's resources.
func launcher() (path, script string, err error) {
	if runtime.GOOS == "windows" {
		return "", "", errors.New("Windows 暂不支持安装命令行工具")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", "", err
	}
	program := filepath.Join(resources, "meetkeep")
	if _, err := os.Stat(program); err != nil {
		return "", "", errors.New("当前版本未包含命令行工具")
	}
	quoted := "'" + strings.ReplaceAll(program, "'", `'\''`) + "'"
	return filepath.Join(home, ".local", "bin", "meetkeep"), fmt.Sprintf("#!/bin/sh\nexec %s \"$@\"\n", quoted), nil
}

// CLI returns where the meetkeep command is, with ~ for the home
// directory, or "" when it is not installed or runs another copy of the
// app.
func (c *Config) CLI() string {
	path, script, err := launcher()
	if err != nil {
		return ""
	}
	if installed, err := os.ReadFile(path); err != nil || string(installed) != script {
		return ""
	}
	return config.Contract(path)
}

// InstallCLI puts the meetkeep command in ~/.local/bin and returns where
// it is, with ~ for the home directory.
func (c *Config) InstallCLI() (string, error) {
	path, script, err := launcher()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return "", err
	}
	return config.Contract(path), nil
}
