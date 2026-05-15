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
	// Test directories: scenarios SHOULD legitimately contain the bad
	// patterns we look for. Static detection on them is pure noise.
	"test":       true,
	"tests":      true,
	"__tests__":  true,
	"__mocks__":  true,
	"testdata":   true,
}

// testFileSuffixes flag individual files as test code (which should not
// be audited even when they live outside a test directory).
var testFileSuffixes = []string{
	"_test.go",
	".test.ts",
	".test.tsx",
	".test.js",
	".test.jsx",
	".spec.ts",
	".spec.tsx",
	".spec.js",
	".spec.jsx",
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
		if isTestFile(path) {
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

func isTestFile(path string) bool {
	base := filepath.Base(path)
	for _, suf := range testFileSuffixes {
		if strings.HasSuffix(base, suf) {
			return true
		}
	}
	return false
}
