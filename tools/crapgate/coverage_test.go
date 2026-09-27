package main

import (
	"go/token"
	"strings"
	"testing"
)

func TestChangedFileDiscoveryReportsInvalidHistory(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Run() {}\n")
	commitTestRepository(t, repo, "base")
	ref := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		_, _, _, _, err := collectChangedFunctions("missing-base", ref)
		if err == nil || !strings.Contains(err.Error(), "list changed Go files") {
			t.Fatalf("invalid history error = %v", err)
		}
	})
}

func TestNestedModuleDiscoveryReportsInvalidRevision(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Run() {}\n")
	commitTestRepository(t, repo, "base")
	withWorkingDirectory(t, repo, func() {
		_, err := pathInScannedModule("missing-ref", "pkg/service.go")
		if err == nil || !strings.Contains(err.Error(), "check nested module") {
			t.Fatalf("invalid nested-module revision error = %v", err)
		}
	})
}

func TestFormatterRejectsMissingASTNode(t *testing.T) {
	if _, err := formatNode(token.NewFileSet(), nil); err == nil {
		t.Fatal("missing AST node accepted")
	}
}
