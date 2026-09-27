package main

import (
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
		_, _, _, _, _, err := collectChangedFunctions(baseRef, headRef)
		if err == nil || !strings.Contains(err.Error(), "left the default build") {
			t.Fatalf("scope change error = %v", err)
		}
	})
}
