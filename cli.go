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
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", "", err
	}
	if runtime.GOOS == "windows" {
		return windowsLauncher(filepath.Join(resources, "meetkeep-cli.exe"))
	}
	home, err := os.UserHomeDir()
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

// windowsLauncher is the launcher on Windows: a batch file in the folder
// of the user that Windows has on the path. The file is read in the code
// page of the console, so the user's folders are named by their
// variables, which keeps a name outside ASCII out of it.
func windowsLauncher(program string) (path, script string, err error) {
	if _, err := os.Stat(program); err != nil {
		return "", "", errors.New("当前版本未包含命令行工具")
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", "", errors.New("找不到 %LOCALAPPDATA%")
	}
	for _, name := range []string{"LOCALAPPDATA", "USERPROFILE"} {
		folder := os.Getenv(name)
		if rest, ok := strings.CutPrefix(program, folder+`\`); ok && folder != "" {
			program = "%" + name + `%\` + rest
			break
		}
	}
	return filepath.Join(local, "Microsoft", "WindowsApps", "meetkeep.cmd"), fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", program), nil
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

// InstallCLI puts the meetkeep command on the user's path, in ~/.local/bin
// or the folder Windows keeps for the user's commands, and returns where
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
