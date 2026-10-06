package main

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// Same published palette location as Widget Core; read only, bounded, no TOML
// evaluation. Explicit reload avoids a polling timer in the idle comparison.
var themeLine = regexp.MustCompile(`(?m)^\s*(foreground|background|accent|red)\s*=\s*["'](#[0-9a-fA-F]{6})["']\s*(?:#.*)?$`)

func loadTheme() map[string]string {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, ".local", "state")
	}
	f, err := os.Open(filepath.Join(root, "omarchy", "current", "theme", "colors.toml"))
	if err != nil {
		return nil
	}
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(body) > 16384 {
		return nil
	}
	result := map[string]string{}
	for _, m := range themeLine.FindAllSubmatch(body, -1) {
		result[string(m[1])] = string(m[2])
	}
	return result
}
