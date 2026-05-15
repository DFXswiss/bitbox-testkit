package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// sourceExtensions are scanned by default. Add more as new wallet stacks emerge.
var sourceExtensions = []string{".go", ".ts", ".tsx", ".js", ".jsx"}

// skipDirs prevents scanning into vendored / generated / heavy paths.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".git":         true,
}

func absPath(p string) (string, error) {
	return filepath.Abs(p)
}

func enumerateSources(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return fs.SkipDir
			}
			if strings.HasPrefix(name, ".") && name != "." {
				return fs.SkipDir
			}
			return nil
		}
		if !hasSourceExtension(path) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func hasSourceExtension(path string) bool {
	for _, ext := range sourceExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
