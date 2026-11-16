//go:build ignore

// Run from the repository root with go run ./scripts/go-format.go [--write].
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	write := len(os.Args) == 2 && os.Args[1] == "--write"
	failed := false
	err := filepath.WalkDir("backend", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Git may check out CRLF on Windows; gofmt checks syntax and layout.
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		formatted, err := format.Source(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if string(data) == string(formatted) {
			return nil
		}
		if write {
			return os.WriteFile(path, formatted, 0644)
		}
		fmt.Println(path)
		failed = true
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if failed {
		os.Exit(1)
	}
}
