// Package main tests the CRAP gate against temporary Git histories and reports.
package main

import (
	"encoding/json"
	"io"
	"log"
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
	if err := execute(nil); err == nil {
		t.Fatal("CLI accepted missing flags")
	}
	if _, err := parseOptions(append(args, "--unknown")); err == nil {
		t.Fatal("unknown flag accepted")
	}
	if _, err := parseOptions([]string{"--base-report"}); err == nil {
		t.Fatal("flag missing its value accepted")
	}
	if _, err := parseOptions([]string{"--base-report", "base.json", "--head-report", "head.json", "--base-ref", "main", "--head-ref"}); err == nil {
		t.Fatal("final flag missing its value accepted")
	}
}

func TestCLIEntrypointReportsExactRevision(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() {}\n")
	commitTestRepository(t, repo, "source")
	ref := gitTestOutput(t, repo, "rev-parse", "HEAD")
	reportPath := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
	withWorkingDirectory(t, repo, func() {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := reader.Close(); err != nil {
				t.Error(err)
			}
		}()
		previousArgs, previousOutput, previousLogFlags := os.Args, os.Stdout, log.Flags()
		defer func() {
			os.Args, os.Stdout = previousArgs, previousOutput
			log.SetFlags(previousLogFlags)
		}()
		os.Args = []string{"crapgate", "--base-report", reportPath, "--head-report", reportPath, "--base-ref", ref, "--head-ref", ref}
		os.Stdout = writer
		main()
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		output, err := io.ReadAll(reader)
		if err != nil || !strings.Contains(string(output), "worst sample/pkg/service.go:Stable") {
			t.Fatalf("CLI output = %q, err=%v", output, err)
		}
	})
}

func TestRunComparesChangesInTemporaryGitRepository(t *testing.T) {
	repo := initTestRepository(t)
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 1 }\n")
	commitTestRepository(t, repo, "base")
	baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
	writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() int { return 2 }\n")
	writeTestFile(t, repo, "pkg/new.go", "package sample\nfunc Added() int { return 3 }\n")
	writeTestFile(t, repo, "pkg/windows_windows.go", "package sample\nfunc WindowsOnly() int { return 4 }\n")
	writeTestFile(t, repo, "pkg/tools.go", "//go:build tools\n\npackage sample\nfunc ToolsOnly() int { return 5 }\n")
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
		if err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: "bad-ref", headRef: ref}, io.Discard); err == nil {
			t.Fatal("invalid base ref accepted")
		}
		if err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: ref, headRef: "bad-ref"}, io.Discard); err == nil {
			t.Fatal("invalid head ref accepted")
		}
		if err := run(options{baseReport: filepath.Join(repo, "missing.json"), headReport: reportPath, baseRef: ref, headRef: ref}, io.Discard); err == nil {
			t.Fatal("missing base report accepted")
		}
		if err := run(options{baseReport: reportPath, headReport: filepath.Join(repo, "missing.json"), baseRef: ref, headRef: ref}, io.Discard); err == nil {
			t.Fatal("missing head report accepted")
		}
		badReport := writeTestReport(t, report{Version: "wrong", Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
		if err := run(options{baseReport: badReport, headReport: reportPath, baseRef: ref, headRef: ref}, io.Discard); err == nil {
			t.Fatal("invalid report accepted")
		}
	})
	if err := writeResults(io.Discard, []result{{Kind: "new", ID: testID("pkg/new.go", "", "Added"), Head: 11, Allowed: false}}); err == nil {
		t.Fatal("policy violation passed")
	}
}

func TestRunReturnsWriterAndChangedSourceErrors(t *testing.T) {
	t.Run("writer failure", func(t *testing.T) {
		repo := initTestRepository(t)
		writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() {}\n")
		commitTestRepository(t, repo, "source")
		ref := gitTestOutput(t, repo, "rev-parse", "HEAD")
		reportPath := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
		withWorkingDirectory(t, repo, func() {
			err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: ref, headRef: ref}, failingWriter{})
			if err == nil || !strings.Contains(err.Error(), "write worst CRAP function") {
				t.Fatalf("run writer error = %v", err)
			}
		})
	})
	t.Run("malformed changed source", func(t *testing.T) {
		repo := initTestRepository(t)
		writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable() {}\n")
		commitTestRepository(t, repo, "base source")
		baseRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
		writeTestFile(t, repo, "pkg/service.go", "package sample\nfunc Stable( {\n")
		commitTestRepository(t, repo, "malformed source")
		headRef := gitTestOutput(t, repo, "rev-parse", "HEAD")
		reportPath := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{reportEntry("pkg/service.go", "Stable", 1)}})
		withWorkingDirectory(t, repo, func() {
			err := run(options{baseReport: reportPath, headReport: reportPath, baseRef: baseRef, headRef: headRef}, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "parse") {
				t.Fatalf("run malformed source error = %v", err)
			}
		})
	})
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
		if err := run(options{baseReport: baseReport, headReport: headReport, baseRef: baseRef, headRef: headRef}, io.Discard); err != nil {
			t.Fatalf("test-only CRAP comparison: %v", err)
		}
		worseLegacy := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 9), reportEntry("pkg/service.go", "Legacy", 15.01),
		}})
		if err := run(options{baseReport: baseReport, headReport: worseLegacy, baseRef: baseRef, headRef: headRef}, io.Discard); err == nil {
			t.Fatal("worsened legacy CRAP accepted for test-only diff")
		}
		aboveLimit := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 10.01), reportEntry("pkg/service.go", "Legacy", 15),
		}})
		if err := run(options{baseReport: baseReport, headReport: aboveLimit, baseRef: baseRef, headRef: headRef}, io.Discard); err == nil {
			t.Fatal("test-only CRAP above 10 accepted")
		}
		partial := writeTestReport(t, report{Version: reportSchemaVersion, Entries: []entry{
			reportEntry("pkg/service.go", "Stable", 9),
		}})
		if err := run(options{baseReport: baseReport, headReport: partial, baseRef: baseRef, headRef: headRef}, io.Discard); err == nil {
			t.Fatal("partial test-only function inventory accepted")
		}
	})
}

func TestTestOnlyAssessmentRejectsEqualCountMismatchedInventory(t *testing.T) {
	baseID := testID("pkg/a.go", "", "A")
	headID := testID("pkg/b.go", "", "B")
	base := map[functionID]entry{baseID: reportEntry("pkg/a.go", "A", 5)}
	head := map[functionID]entry{headID: reportEntry("pkg/b.go", "B", 5)}
	if _, err := evaluateTestOnlyCRAPChanges(base, head); err == nil {
		t.Fatal("equal-count but mismatched production inventory accepted")
	}
}

func TestGitHelpersCoverEmptyMalformedAndSourceErrors(t *testing.T) {
	if _, err := gitOutput("not-a-git-command"); err == nil {
		t.Fatal("git command failure ignored")
	}
	if _, err := parseChangedFiles("malformed-line"); err == nil {
		t.Fatal("malformed name-status line accepted")
	}
	if _, err := changedFiles("missing-base-ref", "missing-head-ref"); err == nil {
		t.Fatal("Git diff failure accepted")
	}
	files, err := parseChangedFiles("\nM\tpkg/service.go\nA\tpkg/new.go\n")
	if err != nil || len(files) != 2 || files[1].status != "A" {
		t.Fatalf("changed files = %#v, err=%v", files, err)
	}
	empty, err := parseChangedFiles(" \n")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty changed files = %#v, err=%v", empty, err)
	}
}

func TestSourceReadAndParseFailuresInGitHistory(t *testing.T) {
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
		if err := addFileFunctions(map[functionID]string{}, "HEAD", "missing.go"); err == nil {
			t.Fatal("missing source accepted by function collector")
		}
	})
	baseRepo := initTestRepository(t)
	writeTestFile(t, baseRepo, "pkg/changed.go", "package broken\nfunc Broken( {\n")
	commitTestRepository(t, baseRepo, "malformed base")
	baseRef := gitTestOutput(t, baseRepo, "rev-parse", "HEAD")
	writeTestFile(t, baseRepo, "pkg/changed.go", "package sample\nfunc Fixed() {}\n")
	commitTestRepository(t, baseRepo, "valid head")
	headRef := gitTestOutput(t, baseRepo, "rev-parse", "HEAD")
	withWorkingDirectory(t, baseRepo, func() {
		if err := collectOneFile(changedFile{status: "M", path: "pkg/changed.go"}, baseRef, headRef, map[functionID]string{}, map[functionID]string{}); err == nil || !strings.Contains(err.Error(), "parse") {
			t.Fatalf("malformed base source error = %v", err)
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

func TestBuildConstraintsMatchLinuxDefaultScope(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		source string
		want   bool
	}{
		{name: "ordinary", path: "pkg/service.go", source: "package sample\nfunc Run() {}\n", want: true},
		{name: "windows suffix", path: "pkg/service_windows.go", source: "package sample\nfunc Run() {}\n", want: false},
		{name: "linux suffix", path: "pkg/service_linux.go", source: "package sample\nfunc Run() {}\n", want: true},
		{name: "tools tag", path: "pkg/service.go", source: "//go:build tools\n\npackage sample\nfunc Run() {}\n", want: false},
		{name: "linux tag", path: "pkg/service.go", source: "//go:build linux\n\npackage sample\nfunc Run() {}\n", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fileInLinuxDefaultBuild(tc.path, []byte(tc.source))
			if err != nil || got != tc.want {
				t.Fatalf("in Linux default build = %t, err=%v, want=%t", got, err, tc.want)
			}
		})
	}
}

func TestMalformedBuildConstraintsFailClosed(t *testing.T) {
	_, err := fileInLinuxDefaultBuild("pkg/service.go", []byte("//go:build linux &&\n\npackage sample\nfunc Run() {}\n"))
	if err == nil {
		t.Fatal("malformed Go build constraint accepted")
	}
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

func TestReportRejectsTrailingJSONValue(t *testing.T) {
	data, err := json.Marshal(testReport(reportEntry("pkg/a.go", "Run", 1)))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte(" {}")...)
	if _, err := parseReport(data); err == nil {
		t.Fatal("report with trailing JSON value accepted")
	}
}

func TestFindEntrySkipsUnrelatedCandidatesBeforePathMatch(t *testing.T) {
	wantID := testID("repo/pkg/a.go", "", "Run")
	matchingID := testID("pkg/a.go", "", "Run")
	unrelatedID := testID("pkg/a.go", "Other", "Run")
	matching := testEntry(matchingID, "Run", 3)
	unrelated := testEntry(unrelatedID, "Run", 7)
	entries := map[functionID]entry{unrelatedID: unrelated, matchingID: matching}
	got, ok := findEntry(entries, wantID)
	if !ok || *got.CRAP != 3 {
		t.Fatalf("path match after unrelated candidate = %#v, %t", got, ok)
	}
}

func TestWriteResultsCoversChangedOutput(t *testing.T) {
	var output strings.Builder
	err := writeResults(&output, []result{{Kind: "changed", ID: testID("pkg/a.go", "*Thing", "Run"), Base: 11, Head: 10, Delta: -1, BaseCC: 11, HeadCC: 10, CCDelta: -1, Allowed: true}})
	if err != nil || !strings.Contains(output.String(), "CC base=11 head=10 delta=-1") {
		t.Fatalf("write result = %q, err=%v", output.String(), err)
	}
}

func TestWritersPropagateOutputFailures(t *testing.T) {
	id := testID("pkg/a.go", "", "Run")
	item := testEntry(id, "Run", 2)
	if err := writeWorst(failingWriter{}, map[functionID]entry{id: item}, 1); err == nil {
		t.Fatal("writeWorst ignored writer failure")
	}
	resultItem := result{Kind: "new", ID: id, Head: 2, Allowed: true}
	if err := writeResults(failingWriter{}, []result{resultItem}); err == nil {
		t.Fatal("writeResults ignored writer failure")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, os.ErrClosed
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
