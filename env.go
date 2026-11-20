package main

import (
	"os"
	"strings"
)

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for line := range strings.Lines(string(data)) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, strings.Trim(value, `"'`))
		}
	}
}
