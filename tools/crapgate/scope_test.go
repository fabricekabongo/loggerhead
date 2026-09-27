package main

import (
	"io"
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

func TestRenamedAwayTestStillChecksCoverageRegression(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 1 }\n")
	writeTestFile(t, repo, "pkg/service_test.go", "package sample\nfunc TestStable() { _ = Stable() }\n")
	commitTestRepository(t, repo, "base with test")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(repo, "pkg", "service_test.go")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repo, "pkg/testdata/service.go", "package sample\nfunc TestStable() { _ = Stable() }\n")
	commitTestRepository(t, repo, "move test out of build")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		changed, err := collectChangedFunctions(baseRef, headRef)
		if err != nil || len(changed.base) != 0 || len(changed.head) != 0 {
			t.Fatalf("renamed-away test inventory = %#v, err=%v", changed, err)
		}
		baseReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 11)}})
		headReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 12)}})
		err = run(options{baseReport: baseReport, headReport: headReport, baseRef: baseRef, headRef: headRef}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "CRAP policy failed") {
			t.Fatalf("test coverage regression after rename error = %v", err)
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

func TestRemovingGoldenFixtureModuleBoundaryFailsClosed(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "quality/go-crap-fixtures/go.mod", "module example.com/fixture\n\ngo 1.23\n")
	writeTestFile(t, repo, "quality/go-crap-fixtures/fixture.go", "package fixture\nfunc Golden() {}\n")
	commitTestRepository(t, repo, "nested fixture")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(repo, "quality", "go-crap-fixtures", "go.mod")); err != nil {
		t.Fatal(err)
	}
	commitTestRepository(t, repo, "remove fixture boundary")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		_, err := changedFiles(baseRef, headRef)
		if err == nil || !strings.Contains(err.Error(), "nested module boundary changed") {
			t.Fatalf("fixture boundary removal error = %v", err)
		}
	})
}
