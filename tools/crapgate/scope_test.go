package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModifiedFileLeavingDefaultBuildFailsClosed(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() {}\n")
	commitTestRepository(t, repo, "base source")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	writeTestFile(t, repo, "pkg/service.go", "//go:build tools\n\npackage sample\nfunc Stable() {}\n")
	commitTestRepository(t, repo, "exclude source from default build")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		_, err := collectChangedFunctions(baseRef, headRef)
		if err == nil || !strings.Contains(err.Error(), "left the default build") {
			t.Fatalf("scope change error = %v", err)
		}
	})
}

func TestRenamedModuleBoundaryChecksDestinationSources(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/old/go.mod", "module example.com/nested\n\ngo 1.23\n")
	writeTestFile(t, repo, "pkg/new/service.go", "package sample\nfunc Stable() {}\n")
	commitTestRepository(t, repo, "base sources")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(repo, "pkg", "old", "go.mod")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repo, "pkg/new/go.mod", "module example.com/nested\n\ngo 1.23\n")
	commitTestRepository(t, repo, "move module boundary")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		_, err := changedFiles(baseRef, headRef)
		if err == nil || !strings.Contains(err.Error(), "nested module boundary changed") {
			t.Fatalf("renamed module boundary error = %v", err)
		}
	})
}
