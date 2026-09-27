// Package main tests the CRAP gate against temporary Git histories and reports.
package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOptionsHandlesValidMissingAndInvalidFlags(t *testing.T) {
	args := []string{"--base-report", "base.json", "--head-report", "head.json", "--base-ref", "main", "--head-ref", "HEAD"}
	parsed, err := parseOptions(args)
	if err != nil || parsed.baseReport != "base.json" || parsed.headRef != "HEAD" {
		t.Fatalf("parsed options = %#v, err=%v", parsed, err)
	}
	if _, err := parseOptions(nil); err == nil {
		t.Fatal("missing flags accepted")
	}
	if _, err := parseOptions(append(args, "--unknown")); err == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestRunComparesChangesInTemporaryGitRepository(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 1 }\n")
	commitTestRepository(t, repo, "base")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 2 }\n")
	writeTestFile(t, repo, "pkg/new.go", "package sample\nfunc Added() int { return 3 }\n")
	writeTestFile(t, repo, "pkg/service_test.go", "package sample\nfunc TestOnly() {}\n")
	commitTestRepository(t, repo, "head")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	baseReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
	headReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
		reportEntry("pkg/service.go", "Stable", 1), reportEntry("pkg/new.go", "Added", 1),
	}})
	withWorkingDirectory(t, repo, func() {
		args := []string{"--base-report", baseReport, "--head-report", headReport, "--base-ref", baseRef, "--head-ref", headRef}
		if err := execute(args); err != nil {
			t.Fatalf("execute comparator: %v", err)
		}
	})
}

func TestRunRejectsRefsReportsAndPolicyViolations(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 1 }\n")
	commitTestRepository(t, repo, "base")
	ref := gitTestOutput(t, repo, "rev-parse", "HEAD")
	reportPath := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
	withWorkingDirectory(t, repo, func() {
		if err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: "bad-ref", headRef: ref}); err == nil {
			t.Fatal("invalid base ref accepted")
		}
		if err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: ref, headRef: "bad-ref"}); err == nil {
			t.Fatal("invalid head ref accepted")
		}
		if err := run(options{baseReport: filepath.Join(repo, "missing.json"), headReport: reportPath, baseRef: ref, headRef: ref}); err == nil {
			t.Fatal("missing base report accepted")
		}
		if err := run(options{baseReport: reportPath, headReport: filepath.Join(repo, "missing.json"), baseRef: ref, headRef: ref}); err == nil {
			t.Fatal("missing head report accepted")
		}
		badReport := writeTestReport(t, report{Version: "wrong", Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
		if err := run(options{baseReport: badReport, headReport: reportPath, baseRef: ref, headRef: ref}); err == nil {
			t.Fatal("invalid report accepted")
		}
	})
	if err := writeResults(io.Discard, []result{{Kind: "new", ID: testID("pkg/new.go", "", "Added"), Head: 11, Allowed: false}}); err == nil {
		t.Fatal("policy violation passed")
	}
}

func TestRunAssessesCRAPChangesForTestOnlyDiff(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 1 }\nfunc Legacy() int { return 2 }\n")
	writeTestFile(t, repo, "pkg/service_test.go", "package sample\nfunc TestStable() { _ = Stable() }\n")
	commitTestRepository(t, repo, "base")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	writeTestFile(t, repo, "pkg/service_test.go", "package sample\nfunc TestStable() { if Stable() != 1 { panic(\"bad\") } }\n")
	commitTestRepository(t, repo, "tests only")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	baseReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
		reportEntry("pkg/service.go", "Stable", 8), reportEntry("pkg/service.go", "Legacy", 15),
	}})
	headReport := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
		reportEntry("pkg/service.go", "Stable", 9), reportEntry("pkg/service.go", "Legacy", 15),
	}})
	withWorkingDirectory(t, repo, func() {
		files, err := changedFiles(baseRef, headRef)
		if err != nil || len(files) != 1 || !strings.HasSuffix(files[0].path, "_test.go") {
			t.Fatalf("test-only file diff = %#v, err=%v", files, err)
		}
		if err := run(options{baseReport: baseReport, headReport: headReport, baseRef: baseRef, headRef: headRef}); err != nil {
			t.Fatalf("test-only CRAP comparison: %v", err)
		}
		worseLegacy := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 9), reportEntry("pkg/service.go", "Legacy", 15.01),
		}})
		if err := run(options{baseReport: baseReport, headReport: worseLegacy, baseRef: baseRef, headRef: headRef}); err == nil {
			t.Fatal("worsened legacy CRAP accepted for test-only diff")
		}
		aboveLimit := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 10.01), reportEntry("pkg/service.go", "Legacy", 15),
		}})
		if err := run(options{baseReport: baseReport, headReport: aboveLimit, baseRef: baseRef, headRef: headRef}); err == nil {
			t.Fatal("test-only CRAP above 10 accepted")
		}
		partial := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 9),
		}})
		if err := run(options{baseReport: baseReport, headReport: partial, baseRef: baseRef, headRef: headRef}); err == nil {
			t.Fatal("partial test-only function inventory accepted")
		}
	})
}

func TestGitHelpersCoverEmptyMalformedAndSourceErrors(t *testing.T) {
	if _, err := gitOutput("not-a-git-command"); err == nil {
		t.Fatal("git command failure ignored")
	}
	if _, err := parseChangedFiles("malformed-line"); err == nil {
		t.Fatal("malformed name-status line accepted")
	}
	files, err := parseChangedFiles("\nM\tpkg/service.go\nA\tpkg/new.go\n")
	if err != nil || len(files) != 2 || files[1].status != "A" {
		t.Fatalf("changed files = %#v, err=%v", files, err)
	}
	empty, err := parseChangedFiles(" \n")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty changed files = %#v, err=%v", empty, err)
	}
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/broken.go", "package broken\nfunc Broken( {\n")
	commitTestRepository(t, repo, "broken syntax")
	withWorkingDirectory(t, repo, func() {
		if err := addFileFunctions(map[functionID]string{}, "HEAD", "pkg/broken.go"); err == nil || !strings.Contains(err.Error(), "parse") {
			t.Fatalf("invalid source error = %v", err)
		}
		if _, err := gitOutput("show", "HEAD:missing.go"); err == nil {
			t.Fatal("missing source accepted")
		}
	})
}

func TestGoldenFixtureIsOutsideProductionScope(t *testing.T) {
	baseFunctions := map[functionID]string{}
	headFunctions := map[functionID]string{}
	fixture := changedFile{status: "A", path: "quality/go-crap-fixtures/fixture.go"}
	if err := collectOneFile(fixture, "missing-base", "missing-head", baseFunctions, headFunctions); err != nil {
		t.Fatalf("nested fixture must not enter production comparison: %v", err)
	}
	if len(baseFunctions) != 0 || len(headFunctions) != 0 {
		t.Fatalf("nested fixture entered production scope: base=%v head=%v", baseFunctions, headFunctions)
	}
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/production.go", "package sample\nfunc Stable() {}\n")
	commitTestRepository(t, repo, "base")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	writeTestFile(t, repo, "quality/go-crap-fixtures/fixture.go", "package fixture\nfunc Fixture() {}\n")
	commitTestRepository(t, repo, "fixture only")
	headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	withWorkingDirectory(t, repo, func() {
		base, head, testOnly, err := collectChangedFunctions(baseRef, headRef)
		if err != nil || !testOnly || len(base) != 0 || len(head) != 0 {
			t.Fatalf("fixture-only diff classification = base %v head %v testOnly %t err %v", base, head, testOnly, err)
		}
	})
}

func TestReportIdentityRejectsMalformedFunctionNames(t *testing.T) {
	for _, item := range []entry{
		{File: "pkg/a.go", Package: "sample", Function: "Counter.Run"},
		{File: "pkg/a.go", Package: "sample", Function: "Run", Receiver: "Counter"},
		{File: "pkg/a.go", Package: "sample", Function: "Counter.", Receiver: "Counter"},
		{Package: "sample", Function: "Run"},
	} {
		if _, err := idFromEntry(item); err == nil {
			t.Fatalf("malformed function identity accepted: %#v", item)
		}
	}
}

func TestWriteResultsCoversChangedOutput(t *testing.T) {
	var output strings.Builder
	err := writeResults(&output, []result{{Kind: "changed", ID: testID("pkg/a.go", "*Thing", "Run"), Base: 11, Head: 10, Delta: -1, BaseCC: 11, HeadCC: 10, CCDelta: -1, Allowed: true}})
	if err != nil || !strings.Contains(output.String(), "CC base=11 head=10 delta=-1") {
		t.Fatalf("write result = %q, err=%v", output.String(), err)
	}
}

func TestWorstSummaryRanksFunctionsWithStatementCoverage(t *testing.T) {
	low := testID("pkg/a.go", "", "Low")
	high := testID("pkg/b.go", "*Counter", "High")
	entries := map[functionID]entry{
		low:  testEntry(low, "Low", 2),
		high: testEntry(high, "*Counter.High", 12),
	}
	var output strings.Builder
	if err := writeWorst(&output, entries, 2); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "*Counter.High") || !strings.Contains(lines[0], "statement-coverage=0%") {
		t.Fatalf("worst function order/coverage = %q", output.String())
	}
}

func reportEntry(file, function string, score float64) entry {
	return entry{File: file, Package: "sample", Function: function, CRAP: floatPtr(score), Cyclomatic: intPtr(1), Coverage: floatPtr(100), Line: intPtr(2)}
}

func writeTestReport(t *testing.T, value report) string {
	t.Helper()
	total := len(value.Entries)
	value.Summary.TotalFunctions = &total
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func initTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGitTest(t, root, "init", "--quiet")
	runGitTest(t, root, "config", "user.email", "crapgate@example.test")
	runGitTest(t, root, "config", "user.name", "CRAP Gate Test")
	return root
}

func writeTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func commitTestRepository(t *testing.T, root, message string) {
	t.Helper()
	runGitTest(t, root, "add", ".")
	runGitTest(t, root, "commit", "--quiet", "-m", message)
}

func gitTestOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	return strings.TrimSpace(runGitTest(t, root, args...))
}

func runGitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s (%v)", args, output, err)
	}
	return string(output)
}

func withWorkingDirectory(t *testing.T, directory string, run func()) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	run()
}
